package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"workout-app/backend/internal/config"
)

const shutdownTimeout = 10 * time.Second

type Server struct {
	cfg     *config.Config
	handler http.Handler
}

func NewServer(cfg *config.Config, handler http.Handler) *Server {
	return &Server{cfg: cfg, handler: handler}
}

func (s *Server) Run() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      s.handler,
		ReadTimeout:  s.cfg.Server.ReadTimeout,
		WriteTimeout: s.cfg.Server.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("%s listening on http://localhost:%d", s.cfg.ServiceName, s.cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-errCh:
		return err
	case <-quit:
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return httpServer.Shutdown(ctx)
}
