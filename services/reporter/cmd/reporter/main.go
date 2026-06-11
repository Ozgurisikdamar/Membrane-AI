// Command reporter is the composition root for the reporter service: it
// consumes verdicts and fans them out to the configured notifiers.
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
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/adapters/kafkabus"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/adapters/memorylog"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/adapters/notify"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/config"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/ports"
)

func main() {
	log := logging.New(os.Getenv("MEMBRANE_REPORTER_LOG_LEVEL"), os.Stdout)
	if err := run(log); err != nil {
		log.Error("reporter exited with error", "err", err)
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
		ServiceName: "reporter", ServiceVersion: "0.1.0",
		OTLPEndpoint: cfg.OTLPEndpoint, Insecure: cfg.OTLPInsecure,
	})
	if err != nil {
		return err
	}
	defer observability.Stop(otelShutdown)

	notifiers := []ports.Notifier{notify.NewLogger(log)}
	if cfg.WebhookURL != "" {
		notifiers = append(notifiers, notify.NewWebhook(cfg.WebhookURL))
		log.Info("webhook notifier enabled")
	}
	if cfg.GitHubToken != "" {
		notifiers = append(notifiers,
			notify.NewGitHubStatus(cfg.GitHubAPIURL, cfg.GitHubToken),
			notify.NewGitHubPRComment(cfg.GitHubAPIURL, cfg.GitHubToken),
		)
		log.Info("github notifiers enabled (commit status + PR comment)")
	}
	dispatch := app.NewDispatchVerdict(memorylog.New(), notifiers...)

	consumer, err := kafkabus.NewConsumer(cfg.KafkaBrokers, cfg.VerdictTopic, cfg.ConsumerGroup, dispatch, log)
	if err != nil {
		return err
	}
	defer consumer.Close()

	healthH := health.New(2 * time.Second)
	healthH.Register("kafka", consumer.Ping)
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
	go func() {
		log.Info("consuming verdicts", "topic", cfg.VerdictTopic, "group", cfg.ConsumerGroup)
		errc <- consumer.Run(ctx)
	}()

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
	log.Info("reporter stopped cleanly")
	return nil
}
