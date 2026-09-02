package identity

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxUserAgentLength = 512
	maxIPLength        = 64
)

type clientKey struct{}

// ClientInfo is request metadata stored with a login session. It is not a secret.
type ClientInfo struct {
	UserAgent string
	IP        string
}

func WithClient(ctx context.Context, info ClientInfo) context.Context {
	info.UserAgent = clampText(info.UserAgent, maxUserAgentLength)
	info.IP = clampText(info.IP, maxIPLength)
	return context.WithValue(ctx, clientKey{}, info)
}

func ClientFrom(ctx context.Context) ClientInfo {
	info, _ := ctx.Value(clientKey{}).(ClientInfo)
	return info
}

// LoginSession is an issued cookie session. TokenHash is the hashed cookie, never the raw token.
type LoginSession struct {
	ID        string
	TokenHash string
	UserID    string
	UserAgent string
	IP        string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// SessionPublicView is the account-settings payload. It must not include the
// cookie, token hash, or IP address.
type SessionPublicView struct {
	ID        string    `json:"id"`
	Device    string    `json:"device"`
	Current   bool      `json:"current"`
	CreatedAt time.Time `json:"created_at"`
}

func (s LoginSession) PublicView(currentTokenHash string) SessionPublicView {
	created := s.CreatedAt.UTC()
	if created.IsZero() {
		created = s.ExpiresAt.UTC()
	}
	return SessionPublicView{
		ID:        s.ID,
		Device:    DeviceLabel(s.UserAgent),
		Current:   s.TokenHash != "" && s.TokenHash == currentTokenHash,
		CreatedAt: created,
	}
}

func DeviceLabel(userAgent string) string {
	ua := strings.ToLower(userAgent)
	if strings.TrimSpace(userAgent) == "" {
		return "Неизвестное устройство"
	}
	browser := "Браузер"
	switch {
	case strings.Contains(ua, "edg/"):
		browser = "Edge"
	case strings.Contains(ua, "chrome/") && !strings.Contains(ua, "edg/"):
		browser = "Chrome"
	case strings.Contains(ua, "firefox/"):
		browser = "Firefox"
	case strings.Contains(ua, "safari/"):
		browser = "Safari"
	}
	switch {
	case strings.Contains(ua, "iphone"):
		return browser + " на iPhone"
	case strings.Contains(ua, "ipad"):
		return browser + " на iPad"
	case strings.Contains(ua, "android"):
		return browser + " на Android"
	case strings.Contains(ua, "mac os") || strings.Contains(ua, "macintosh"):
		return browser + " на Mac"
	case strings.Contains(ua, "windows"):
		return browser + " на Windows"
	case strings.Contains(ua, "linux"):
		return browser + " на Linux"
	}
	return browser
}

func clampText(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || utf8.RuneCountInString(value) <= max {
		return value
	}
	runes := []rune(value)
	return string(runes[:max])
}
