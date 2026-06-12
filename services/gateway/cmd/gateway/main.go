// Command gateway is the composition root for the MCP gateway: it loads the
// governance policy, wires the audit sink and use-cases, and runs the HTTP +
// health servers with graceful shutdown.
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

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/health"
	"github.com/Ozgurisikdamar/Membrane-AI/pkg/logging"
	"github.com/Ozgurisikdamar/Membrane-AI/pkg/observability"
	"github.com/Ozgurisikdamar/Membrane-AI/services/gateway/internal/adapters/httpapi"
	"github.com/Ozgurisikdamar/Membrane-AI/services/gateway/internal/adapters/logaudit"
	"github.com/Ozgurisikdamar/Membrane-AI/services/gateway/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/gateway/internal/config"
)

func main() {
	log := logging.New(os.Getenv("MEMBRANE_GATEWAY_LOG_LEVEL"), os.Stdout)
	if err := run(log); err != nil {
		log.Error("gateway exited with error", "err", err)
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
		ServiceName: "gateway", ServiceVersion: "0.1.0",
		OTLPEndpoint: cfg.OTLPEndpoint, Insecure: cfg.OTLPInsecure,
	})
	if err != nil {
		log.Warn("tracing setup failed; continuing without tracing", "err", err)
	}
	defer observability.Stop(otelShutdown)

	gw := app.NewGateway(cfg.Policy, logaudit.New(log))
	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           otelhttp.NewHandler(httpapi.NewMux(gw), "gateway"),
		ReadHeaderTimeout: 5 * time.Second,
	}

	healthH := health.New(2 * time.Second)
	healthSrv := &http.Server{Addr: cfg.HealthAddr, Handler: healthH.Mux(), ReadHeaderTimeout: 5 * time.Second}

	errc := make(chan error, 2)
	go func() {
		log.Info("gateway listening", "addr", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
			return
		}
		errc <- nil
	}()
	go func() {
		log.Info("health listening", "addr", cfg.HealthAddr)
		if err := healthSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
			return
		}
		errc <- nil
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down")
	case err := <-errc:
		if err != nil {
			return err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	_ = healthSrv.Shutdown(shutdownCtx)
	return nil
}
