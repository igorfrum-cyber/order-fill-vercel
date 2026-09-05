package grpcutil

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"
)

func Serve(ctx context.Context, grpcAddr, healthAddr string, grpcSrv *grpc.Server, health http.Handler) error {
	ln, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return err
	}
	httpSrv := &http.Server{Addr: healthAddr, Handler: health, ReadHeaderTimeout: 5 * time.Second}
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
		grpcSrv.GracefulStop()
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		grpcSrv.GracefulStop()
		return httpSrv.Shutdown(shutdownCtx)
	}
}
