package objectstore

import "testing"

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantHost string
		wantSSL  bool
	}{
		{name: "http url", endpoint: "http://minio:9000", wantHost: "minio:9000", wantSSL: false},
		{name: "https url", endpoint: "https://storage.example.com", wantHost: "storage.example.com", wantSSL: true},
		{name: "https url with port", endpoint: "https://storage.example.com:443", wantHost: "storage.example.com:443", wantSSL: true},
		{name: "bare host", endpoint: "minio:9000", wantHost: "minio:9000", wantSSL: false},
		{name: "bare host is trimmed", endpoint: " minio:9000 ", wantHost: "minio:9000", wantSSL: false},
		{name: "trailing slash", endpoint: "http://minio:9000/", wantHost: "minio:9000", wantSSL: false},
		{name: "bare host trailing slash", endpoint: "minio:9000/", wantHost: "minio:9000", wantSSL: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			host, useSSL, err := parseEndpoint(test.endpoint)
			if err != nil {
				t.Fatalf("parse endpoint: %v", err)
			}
			if host != test.wantHost {
				t.Fatalf("host: got %q want %q", host, test.wantHost)
			}
			if useSSL != test.wantSSL {
				t.Fatalf("useSSL: got %v want %v", useSSL, test.wantSSL)
			}
		})
	}
}

func TestParseEndpointErrors(t *testing.T) {
	for _, endpoint := range []string{"", "   ", "ftp://minio:9000", "http://"} {
		if _, _, err := parseEndpoint(endpoint); err == nil {
			t.Fatalf("expected an error for endpoint %q", endpoint)
		}
	}
}

func TestNewStoreRequiresBucket(t *testing.T) {
	if _, err := NewStore("http://minio:9000", "key", "secret", " "); err == nil {
		t.Fatal("expected an error for an empty bucket")
	}
}

func TestNewStoreRejectsBadEndpoint(t *testing.T) {
	if _, err := NewStore("ftp://minio:9000", "key", "secret", "bucket"); err == nil {
		t.Fatal("expected an error for an unsupported scheme")
	}
}
