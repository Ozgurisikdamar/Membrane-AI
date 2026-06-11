// Command resolver is the composition root for the context resolver service.
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
	resolverv1 "github.com/Ozgurisikdamar/Membrane-AI/proto/gen/membrane/resolver/v1"
	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/adapters/embed"
	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/adapters/grpcserver"
	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/adapters/postgres"
	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/config"
)

func main() {
	log := logging.New(os.Getenv("MEMBRANE_RESOLVER_LOG_LEVEL"), os.Stdout)
	if err := run(log); err != nil {
		log.Error("resolver exited with error", "err", err)
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

	embedder, err := embed.NewStub(cfg.EmbedDim) // swap point: ports.Embedder (D-020)
	if err != nil {
		return err
	}
	index, err := postgres.New(ctx, cfg.DatabaseURL, cfg.EmbedDim)
	if err != nil {
		return err
	}
	defer index.Close()

	resolve := app.NewResolveContext(embedder, index)

	grpcSrv := grpc.NewServer()
	resolverv1.RegisterResolverServiceServer(grpcSrv, grpcserver.New(resolve))

	healthH := health.New(2 * time.Second)
	healthH.Register("postgres", index.Ping)
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
	log.Info("resolver stopped cleanly")
	return nil
}
