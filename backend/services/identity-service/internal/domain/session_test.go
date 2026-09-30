package domain

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestDeviceLabelNamesCommonWorkBrowsers(t *testing.T) {
	cases := map[string]string{
		"": "Неизвестное устройство",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15":                   "Safari на Mac",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36":                         "Chrome на Windows",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0":           "Edge на Windows",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1": "Safari на iPhone",
		"Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Mobile Safari/537.36":                            "Chrome на Android",
		"Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0":                                                                  "Firefox на Linux",
	}
	for ua, want := range cases {
		if got := DeviceLabel(ua); got != want {
			t.Fatalf("DeviceLabel(%q)\n got %q\nwant %q", ua, got, want)
		}
	}
}

func TestSessionPublicViewOmitsNetworkDetails(t *testing.T) {
	view := LoginSession{
		ID:        "sess-1",
		TokenHash: "secret-hash",
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
		IP:        "10.0.0.8",
	}.PublicView("secret-hash")
	if view.ID != "sess-1" || !view.Current || view.Device != "Safari на Mac" {
		t.Fatalf("view %#v", view)
	}
	raw, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if strings.Contains(body, "secret-hash") || strings.Contains(body, "10.0.0.8") {
		t.Fatalf("leaked private fields: %s", body)
	}
}

func TestClientContextTrimsAndBoundsMetadata(t *testing.T) {
	t.Parallel()

	if got := ClientFrom(t.Context()); got != (ClientInfo{}) {
		t.Fatalf("ClientFrom(empty) = %+v", got)
	}
	info := ClientFrom(WithClient(t.Context(), ClientInfo{UserAgent: " agent ", IP: " 10.0.0.1 "}))
	if info.UserAgent != "agent" || info.IP != "10.0.0.1" {
		t.Fatalf("ClientFrom() = %+v", info)
	}
	long := WithClient(t.Context(), ClientInfo{
		UserAgent: strings.Repeat("a", maxUserAgentLength+1),
		IP:        strings.Repeat("1", maxIPLength+1),
	})
	bounded := ClientFrom(long)
	if len(bounded.UserAgent) != maxUserAgentLength || len(bounded.IP) != maxIPLength {
		t.Fatalf("metadata was not bounded: ua=%d ip=%d", len(bounded.UserAgent), len(bounded.IP))
	}
	if got := ClientFrom(WithClient(context.Background(), ClientInfo{})); got != (ClientInfo{}) {
		t.Fatalf("empty client info = %+v", got)
	}
}

func TestDeviceLabelUsesGenericBrowserForUnknownPlatformAndNamesIPad(t *testing.T) {
	t.Parallel()

	if got := DeviceLabel("SomeBrowser/1.0"); got != "Браузер" {
		t.Fatalf("unknown browser label = %q", got)
	}
	if got := DeviceLabel("Mozilla/5.0 (iPad) Version/17.0 Safari/605.1"); got != "Safari на iPad" {
		t.Fatalf("iPad label = %q", got)
	}
}
