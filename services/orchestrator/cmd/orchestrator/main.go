// Command orchestrator is the composition root for the orchestrator service:
// it wires the verdict cache, analysis stages, verdict publisher and the Kafka
// consumer into the ProcessSubmission Saga, and runs the health server with
// graceful shutdown.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/health"
	"github.com/Ozgurisikdamar/Membrane-AI/pkg/logging"
	"github.com/Ozgurisikdamar/Membrane-AI/pkg/observability"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/grpcstage"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/kafkabus"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/memorycache"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/memorypublisher"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/outboxstore"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/rediscache"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/semanticstage"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/stages"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/config"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/ports"
)

// systemClock adapts time.Now to the Clock port.
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

func main() {
	log := logging.New(os.Getenv("MEMBRANE_ORCHESTRATOR_LOG_LEVEL"), os.Stdout)
	if err := run(log); err != nil {
		log.Error("orchestrator exited with error", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	otelShutdown, err := observability.Setup(ctx, observability.Config{
		ServiceName: "orchestrator", ServiceVersion: "0.1.0",
		OTLPEndpoint: cfg.OTLPEndpoint, Insecure: cfg.OTLPInsecure,
	})
	if err != nil {
		log.Warn("tracing setup failed; continuing without tracing", "err", err)
	}
	defer observability.Stop(otelShutdown)

	healthH := health.New(2 * time.Second)

	secretScan := stages.NewSecretScan()
	saga, consumer, relay, cleanup, err := wire(cfg, secretScan, healthH, log)
	if err != nil {
		return err
	}
	defer cleanup()

	healthSrv := &http.Server{Addr: cfg.HealthAddr, Handler: healthH.Mux(), ReadHeaderTimeout: 5 * time.Second}
	errc := make(chan error, 2)

	go func() {
		log.Info("health listening", "addr", cfg.HealthAddr)
		if err := healthSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
			return
		}
		errc <- nil
	}()

	if consumer != nil {
		go func() {
			log.Info("consuming submissions",
				"topic", cfg.SubmissionTopic, "group", cfg.ConsumerGroup, "brokers", cfg.KafkaBrokers)
			errc <- consumer.Run(ctx)
		}()
	} else {
		log.Warn("orchestrator running in in-memory mode (no Kafka consumer) — dev only", "saga_ready", saga != nil)
	}

	if relay != nil {
		go func() {
			log.Info("outbox relay running", "interval", cfg.OutboxInterval, "batch", cfg.OutboxBatch)
			relay.Run(ctx)
		}()
	}

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case e := <-errc:
		if e != nil {
			return e
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	_ = healthSrv.Shutdown(shutdownCtx)
	log.Info("orchestrator stopped cleanly")
	return nil
}

// wire builds the Saga and (in Kafka mode) the consumer and outbox relay;
// returns a cleanup that closes adapters in reverse order.
func wire(cfg config.Config, fallback ports.AnalysisStage, healthH *health.Handler, log *slog.Logger) (*app.ProcessSubmission, *kafkabus.Consumer, *outboxstore.Relay, func(), error) {
	opts := app.Options{
		StageDeadline:    cfg.StageDeadline,
		FallbackDeadline: cfg.FallbackDeadline,
		RulesetVersion:   cfg.RulesetVersion,
	}

	// Stage chain: the remote analyzer when configured (else the in-process
	// secret scan), then the advisory semantic stage when configured. The
	// secret scan always remains the deterministic fallback.
	stagesChain := []ports.AnalysisStage{fallback}
	var stageCleanups []func()
	if cfg.AnalyzerAddr != "" {
		remote, rerr := grpcstage.New(cfg.AnalyzerAddr)
		if rerr != nil {
			return nil, nil, nil, nil, rerr
		}
		stagesChain = []ports.AnalysisStage{remote}
		stageCleanups = append(stageCleanups, func() { _ = remote.Close() })
		log.Info("analyzer stage enabled", "addr", cfg.AnalyzerAddr)
	}
	if cfg.SemanticURL != "" {
		var fetcher semanticstage.ContextFetcher
		if cfg.ResolverAddr != "" {
			rf, rerr := semanticstage.NewResolverFetcher(cfg.ResolverAddr)
			if rerr != nil {
				return nil, nil, nil, nil, rerr
			}
			stageCleanups = append(stageCleanups, func() { _ = rf.Close() })
			fetcher = rf
			log.Info("resolver RAG context enabled", "addr", cfg.ResolverAddr)
		}
		// Advisory tier (D-024): wrapped in Optional so its failure surfaces
		// as a warning finding instead of erasing required-stage findings.
		stagesChain = append(stagesChain, app.Optional(semanticstage.New(cfg.SemanticURL, fetcher)))
		log.Info("semantic stage enabled", "url", cfg.SemanticURL)
	}

	closeStage := func() {
		for _, c := range stageCleanups {
			c()
		}
	}

	if cfg.UseInMemory {
		saga := app.NewProcessSubmission(
			memorycache.New(), stagesChain, fallback, memorypublisher.New(), systemClock{}, opts)
		return saga, nil, nil, closeStage, nil
	}

	cache := rediscache.New(cfg.RedisAddr, cfg.CacheTTL)
	healthH.Register("redis", cache.Ping)

	// Durability first (D-013): the Saga publishes into the transactional
	// outbox (audit + event in one ACID tx); the relay ships pending rows to
	// Kafka asynchronously.
	store, err := outboxstore.New(context.Background(), cfg.DatabaseURL, cfg.VerdictTopic)
	if err != nil {
		_ = cache.Close()
		closeStage()
		return nil, nil, nil, nil, err
	}
	healthH.Register("postgres", store.Ping)

	publisher, err := kafkabus.NewPublisher(cfg.KafkaBrokers, cfg.VerdictTopic)
	if err != nil {
		store.Close()
		_ = cache.Close()
		closeStage()
		return nil, nil, nil, nil, err
	}

	saga := app.NewProcessSubmission(cache, stagesChain, fallback, store, systemClock{}, opts)
	relay := outboxstore.NewRelay(store, publisher.PublishRecord, cfg.OutboxInterval, cfg.OutboxBatch, log)

	consumer, err := kafkabus.NewConsumer(cfg.KafkaBrokers, cfg.SubmissionTopic, cfg.ConsumerGroup, saga, log)
	if err != nil {
		publisher.Close()
		store.Close()
		_ = cache.Close()
		closeStage()
		return nil, nil, nil, nil, err
	}
	healthH.Register("kafka", consumer.Ping)

	cleanup := func() {
		consumer.Close()
		publisher.Close()
		store.Close()
		_ = cache.Close()
		closeStage()
	}
	return saga, consumer, relay, cleanup, nil
}
