package objectstore

import "testing"

func TestParseEndpoint(t *testing.T) {
	t.Parallel()
	host, secure, err := parseEndpoint("http://minio:9000", true)
	if err != nil || host != "minio:9000" || secure {
		t.Fatalf("host=%q secure=%v err=%v", host, secure, err)
	}
	host, secure, err = parseEndpoint("minio:9000", false)
	if err != nil || host != "minio:9000" || secure {
		t.Fatalf("bare host=%q secure=%v err=%v", host, secure, err)
	}
}
