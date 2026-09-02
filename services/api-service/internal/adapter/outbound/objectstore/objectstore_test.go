package objectstore

import "testing"

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		useSSL   bool
		wantHost string
		wantSSL  bool
	}{
		{name: "http url", endpoint: "http://minio:9000", wantHost: "minio:9000", wantSSL: false},
		{name: "http url overrides ssl flag", endpoint: "http://minio:9000", useSSL: true, wantHost: "minio:9000", wantSSL: false},
		{name: "https url", endpoint: "https://x.com", wantHost: "x.com", wantSSL: true},
		{name: "bare host keeps flag", endpoint: "minio:9000", useSSL: true, wantHost: "minio:9000", wantSSL: true},
		{name: "bare host without flag", endpoint: " minio:9000 ", wantHost: "minio:9000", wantSSL: false},
		{name: "trailing slash", endpoint: "http://minio:9000/", wantHost: "minio:9000", wantSSL: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			host, useSSL, err := parseEndpoint(test.endpoint, test.useSSL)
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
		if _, _, err := parseEndpoint(endpoint, false); err == nil {
			t.Fatalf("expected an error for endpoint %q", endpoint)
		}
	}
}

func TestNewStoreRequiresBucket(t *testing.T) {
	if _, err := NewStore("http://minio:9000", "key", "secret", " ", false); err == nil {
		t.Fatal("expected an error for an empty bucket")
	}
}
