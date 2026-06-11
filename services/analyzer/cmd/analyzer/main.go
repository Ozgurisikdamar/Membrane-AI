// Command analyzer is the composition root for the static analyzer service.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/health"
	"github.com/Ozgurisikdamar/Membrane-AI/pkg/logging"
	"github.com/Ozgurisikdamar/Membrane-AI/pkg/observability"
	analyzerv1 "github.com/Ozgurisikdamar/Membrane-AI/proto/gen/membrane/analyzer/v1"
	"github.com/Ozgurisikdamar/Membrane-AI/services/analyzer/internal/adapters/grpcserver"
	"github.com/Ozgurisikdamar/Membrane-AI/services/analyzer/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/analyzer/internal/config"
	"github.com/Ozgurisikdamar/Membrane-AI/services/analyzer/internal/domain"
)

func main() {
	log := logging.New(os.Getenv("MEMBRANE_ANALYZER_LOG_LEVEL"), os.Stdout)
	if err := run(log); err != nil {
		log.Error("analyzer exited with error", "err", err)
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
		ServiceName: "analyzer", ServiceVersion: "0.1.0",
		OTLPEndpoint: cfg.OTLPEndpoint, Insecure: cfg.OTLPInsecure,
	})
	if err != nil {
		log.Warn("tracing setup failed; continuing without tracing", "err", err)
	}
	defer observability.Stop(otelShutdown)

	analyze := app.NewAnalyzeDiff(domain.NewSecretDetector(), domain.NewRiskyPatternDetector())

	grpcSrv := grpc.NewServer()
	analyzerv1.RegisterAnalyzerServiceServer(grpcSrv, grpcserver.New(analyze))

	healthH := health.New(2 * time.Second)
	healthSrv := &http.Server{Addr: cfg.HealthAddr, Handler: healthH.Mux(), ReadHeaderTimeout: 5 * time.Second}

	errc := make(chan error, 2)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return err
	}
	go func() {
		log.Info("grpc listening", "addr", cfg.GRPCAddr)
		errc <- grpcSrv.Serve(lis)
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
		log.Info("shutdown signal received")
	case e := <-errc:
		if e != nil {
			return e
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	grpcSrv.GracefulStop()
	_ = healthSrv.Shutdown(shutdownCtx)
	log.Info("analyzer stopped cleanly")
	return nil
}
