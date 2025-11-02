package httpsrv

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type Server struct {
	srv    *http.Server
	router *http.ServeMux
}

type ServerOptions struct {
	MetricsHandler http.HandlerFunc
	MetricsPath    string
}

func NewServer(addr string, opts ServerOptions) *Server {
	router := http.NewServeMux()

	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	if opts.MetricsPath == "" {
		opts.MetricsPath = "/metrics"
	}

	router.Handle("/health", healthHandler())
	router.Handle(opts.MetricsPath, opts.MetricsHandler)

	return &Server{
		srv:    srv,
		router: router,
	}
}

func (s *Server) ListenAddr() string {
	return s.srv.Addr
}

func (s *Server) Start() error {
	err := s.srv.ListenAndServe()

	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
