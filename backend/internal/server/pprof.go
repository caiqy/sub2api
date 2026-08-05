package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type PprofServer struct {
	server   *http.Server
	listener net.Listener
}

func StartPprofServer(cfg config.PprofConfig) (*PprofServer, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	addr, err := net.ResolveTCPAddr("tcp", cfg.Address)
	if err != nil || addr.IP == nil || !addr.IP.IsLoopback() {
		return nil, fmt.Errorf("pprof address must resolve to a loopback address: %q", cfg.Address)
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen pprof: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	for _, name := range []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"} {
		mux.Handle("/debug/pprof/"+name, pprof.Handler(name))
	}

	httpServer := &http.Server{Handler: mux}
	instance := &PprofServer{server: httpServer, listener: listener}
	go func() {
		if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Pprof server serve failed: %v", err)
		}
	}()
	return instance, nil
}

func (s *PprofServer) Address() string {
	return s.listener.Addr().String()
}

func (s *PprofServer) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if err := s.server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
