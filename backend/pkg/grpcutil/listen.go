package grpcutil

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"
)

const (
	shutdownTimeout = 10 * time.Second
)

func Serve(ctx context.Context, grpcAddr, healthAddr string, grpcSrv *grpc.Server, health http.Handler) error {
	ln, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return err
	}
	httpSrv := &http.Server{
		Addr:              healthAddr,
		Handler:           health,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	errc := make(chan error, 2)
	go func() { errc <- grpcSrv.Serve(ln) }()
	go func() {
		err := httpSrv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
			return
		}
		errc <- nil
	}()
	select {
	case err := <-errc:
		stopGRPC(grpcSrv, shutdownTimeout)
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		stopGRPC(grpcSrv, shutdownTimeout)
		return httpSrv.Shutdown(shutdownCtx)
	}
}

func stopGRPC(s *grpc.Server, timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(done)
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		s.Stop()
		<-done
	}
}
