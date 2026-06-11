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
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/grpcstage"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/kafkabus"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/memorycache"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/memorypublisher"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/rediscache"
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

	healthH := health.New(2 * time.Second)

	secretScan := stages.NewSecretScan()
	saga, consumer, cleanup, err := wire(cfg, secretScan, healthH, log)
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

// wire builds the Saga and (in Kafka mode) the consumer; returns a cleanup that
// closes adapters in reverse order.
func wire(cfg config.Config, fallback ports.AnalysisStage, healthH *health.Handler, log *slog.Logger) (*app.ProcessSubmission, *kafkabus.Consumer, func(), error) {
	opts := app.Options{
		StageDeadline:    cfg.StageDeadline,
		FallbackDeadline: cfg.FallbackDeadline,
		RulesetVersion:   cfg.RulesetVersion,
	}

	// Stage chain: the remote analyzer when configured, else the in-process
	// secret scan. The secret scan always remains the deterministic fallback.
	stagesChain := []ports.AnalysisStage{fallback}
	var stageCleanup func()
	if cfg.AnalyzerAddr != "" {
		remote, rerr := grpcstage.New(cfg.AnalyzerAddr)
		if rerr != nil {
			return nil, nil, nil, rerr
		}
		stagesChain = []ports.AnalysisStage{remote}
		stageCleanup = func() { _ = remote.Close() }
		log.Info("analyzer stage enabled", "addr", cfg.AnalyzerAddr)
	}

	closeStage := func() {
		if stageCleanup != nil {
			stageCleanup()
		}
	}

	if cfg.UseInMemory {
		saga := app.NewProcessSubmission(
			memorycache.New(), stagesChain, fallback, memorypublisher.New(), systemClock{}, opts)
		return saga, nil, closeStage, nil
	}

	cache := rediscache.New(cfg.RedisAddr, cfg.CacheTTL)
	healthH.Register("redis", cache.Ping)

	publisher, err := kafkabus.NewPublisher(cfg.KafkaBrokers, cfg.VerdictTopic)
	if err != nil {
		_ = cache.Close()
		closeStage()
		return nil, nil, nil, err
	}

	saga := app.NewProcessSubmission(cache, stagesChain, fallback, publisher, systemClock{}, opts)

	consumer, err := kafkabus.NewConsumer(cfg.KafkaBrokers, cfg.SubmissionTopic, cfg.ConsumerGroup, saga, log)
	if err != nil {
		publisher.Close()
		_ = cache.Close()
		closeStage()
		return nil, nil, nil, err
	}
	healthH.Register("kafka", consumer.Ping)

	cleanup := func() {
		consumer.Close()
		publisher.Close()
		_ = cache.Close()
		closeStage()
	}
	return saga, consumer, cleanup, nil
}
