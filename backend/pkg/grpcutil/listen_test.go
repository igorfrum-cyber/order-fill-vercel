package grpcutil

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestServeStopsOnContextCancellation(t *testing.T) {
	grpcAddr := unusedAddress(t)
	healthAddr := unusedAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := grpc.NewServer()
	result := make(chan error, 1)
	go func() {
		result <- Serve(ctx, grpcAddr, healthAddr, server, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
	}()

	waitForHealth(t, healthAddr)
	cancel()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Serve() = %v, want nil on cancellation", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Serve did not stop after context cancellation")
	}
}

func TestServeReturnsHTTPListenError(t *testing.T) {
	grpcAddr := unusedAddress(t)
	healthListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = healthListener.Close() }()
	server := grpc.NewServer()
	err = Serve(context.Background(), grpcAddr, healthListener.Addr().String(), server, http.NotFoundHandler())
	if err == nil {
		t.Fatal("Serve() succeeded with an occupied health address")
	}
}

func TestServeReturnsGRPCListenError(t *testing.T) {
	err := Serve(context.Background(), "bad address", "127.0.0.1:0", grpc.NewServer(), http.NotFoundHandler())
	if err == nil {
		t.Fatal("Serve() succeeded with an invalid gRPC address")
	}
}

func TestNewServerUsesLocalInsecureTransport(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("GRPC_TLS_MODE", "insecure")
	server := NewServer()
	server.Stop()
}

func unusedAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return address
}

func waitForHealth(t *testing.T, address string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get("http://" + address)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusNoContent {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("health listener did not become ready")
}
