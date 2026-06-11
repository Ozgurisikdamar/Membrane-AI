// Command ingestion is the composition root for the ingestion service: it loads
// config, constructs adapters, wires them into use-cases, and runs the gRPC,
// webhook and health servers with graceful shutdown.
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
	ingestionv1 "github.com/Ozgurisikdamar/Membrane-AI/proto/gen/membrane/ingestion/v1"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/adapters/grpcserver"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/adapters/httpwebhook"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/adapters/idgen"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/adapters/kafka"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/adapters/memory"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/config"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/ports"
)

func main() {
	log := logging.New(os.Getenv("MEMBRANE_INGESTION_LOG_LEVEL"), os.Stdout)
	if err := run(log); err != nil {
		log.Error("ingestion exited with error", "err", err)
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

	var publisher ports.EventPublisher
	var onShutdown []func()
	if cfg.UseInMemory {
		publisher = memory.NewPublisher()
		log.Warn("ingestion running with in-memory publisher (no Kafka) — dev mode only")
	} else {
		kp, kerr := kafka.NewPublisher(cfg.KafkaBrokers, cfg.KafkaTopic)
		if kerr != nil {
			return kerr
		}
		publisher = kp
		onShutdown = append(onShutdown, kp.Close)
		healthH.Register("kafka", kp.Ping)
	}

	enqueue := app.NewEnqueueSubmission(publisher, idgen.UUID{}, idgen.SystemClock{})

	grpcSrv := grpc.NewServer()
	ingestionv1.RegisterIngestionServiceServer(grpcSrv, grpcserver.New(enqueue))

	webhookMux := http.NewServeMux()
	webhookMux.Handle("/webhook", httpwebhook.New(enqueue, cfg.WebhookSecret))
	httpSrv := &http.Server{Addr: cfg.HTTPAddr, Handler: webhookMux, ReadHeaderTimeout: 5 * time.Second}
	healthSrv := &http.Server{Addr: cfg.HealthAddr, Handler: healthH.Mux(), ReadHeaderTimeout: 5 * time.Second}

	errc := make(chan error, 3)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return err
	}
	go func() {
		log.Info("grpc listening", "addr", cfg.GRPCAddr)
		errc <- grpcSrv.Serve(lis)
	}()
	go func() {
		log.Info("webhook listening", "addr", cfg.HTTPAddr)
		errc <- serveHTTP(httpSrv)
	}()
	go func() {
		log.Info("health listening", "addr", cfg.HealthAddr)
		errc <- serveHTTP(healthSrv)
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
	_ = httpSrv.Shutdown(shutdownCtx)
	_ = healthSrv.Shutdown(shutdownCtx)
	for _, fn := range onShutdown {
		fn()
	}
	log.Info("ingestion stopped cleanly")
	return nil
}

func serveHTTP(s *http.Server) error {
	if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
