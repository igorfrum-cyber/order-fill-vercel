package grpcutil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	envGRPCTLSMode       = "GRPC_TLS_MODE"
	envGRPCTLSCertFile   = "GRPC_TLS_CERT_FILE"
	envGRPCTLSKeyFile    = "GRPC_TLS_KEY_FILE"
	envGRPCTLSCAFile     = "GRPC_TLS_CA_FILE"
	envGRPCTLSServerName = "GRPC_TLS_SERVER_NAME"
)

func clientTransportCredentials(target string) (credentials.TransportCredentials, error) {
	mode, err := grpcTLSMode()
	if err != nil {
		return nil, err
	}
	if mode == "insecure" {
		return insecure.NewCredentials(), nil
	}
	caPool, err := loadCertPool(os.Getenv(envGRPCTLSCAFile), mode == "mtls")
	if err != nil {
		return nil, err
	}
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    caPool,
		ServerName: grpcTLSServerName(target),
	}
	if mode == "mtls" {
		cert, err := loadKeyPair()
		if err != nil {
			return nil, err
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return credentials.NewTLS(cfg), nil
}

func serverTransportOptions() ([]grpc.ServerOption, error) {
	mode, err := grpcTLSMode()
	if err != nil {
		return nil, err
	}
	if mode == "insecure" {
		return nil, nil
	}
	cert, err := loadKeyPair()
	if err != nil {
		return nil, err
	}
	cfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}
	if mode == "mtls" {
		caPool, err := loadCertPool(os.Getenv(envGRPCTLSCAFile), true)
		if err != nil {
			return nil, err
		}
		cfg.ClientCAs = caPool
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return []grpc.ServerOption{grpc.Creds(credentials.NewTLS(cfg))}, nil
}

func grpcTLSMode() (string, error) {
	switch mode := strings.ToLower(strings.TrimSpace(os.Getenv(envGRPCTLSMode))); mode {
	case "", "insecure", "disabled", "off":
		return "insecure", nil
	case "tls", "mtls":
		return mode, nil
	default:
		return "", fmt.Errorf("%s must be one of insecure, tls, mtls", envGRPCTLSMode)
	}
}

func grpcTLSServerName(target string) string {
	if explicit := strings.TrimSpace(os.Getenv(envGRPCTLSServerName)); explicit != "" {
		return explicit
	}
	host, _, err := net.SplitHostPort(target)
	if err == nil {
		return strings.Trim(host, "[]")
	}
	return target
}

func loadKeyPair() (tls.Certificate, error) {
	certFile := strings.TrimSpace(os.Getenv(envGRPCTLSCertFile))
	keyFile := strings.TrimSpace(os.Getenv(envGRPCTLSKeyFile))
	if certFile == "" || keyFile == "" {
		return tls.Certificate{}, fmt.Errorf("%s and %s are required when %s is tls or mtls", envGRPCTLSCertFile, envGRPCTLSKeyFile, envGRPCTLSMode)
	}
	return tls.LoadX509KeyPair(certFile, keyFile)
}

func loadCertPool(path string, required bool) (*x509.CertPool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		if required {
			return nil, fmt.Errorf("%s is required when %s is mtls", envGRPCTLSCAFile, envGRPCTLSMode)
		}
		return nil, nil
	}
	// #nosec G304 -- TLS material paths are operator-owned startup configuration, not request input.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", envGRPCTLSCAFile, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(raw) {
		return nil, fmt.Errorf("%s does not contain PEM certificates", envGRPCTLSCAFile)
	}
	return pool, nil
}
