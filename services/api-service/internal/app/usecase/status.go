package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/identity"
)

const defaultStatusTimeout = 2 * time.Second

type StatusProbe struct {
	ID   string
	Ping func(context.Context) error
}

type Status struct {
	timeout func() time.Duration
	probes  []StatusProbe
}

func NewStatus(timeout func() time.Duration) *Status {
	if timeout == nil {
		timeout = func() time.Duration { return defaultStatusTimeout }
	}
	return &Status{timeout: timeout}
}

func (s *Status) WithProbes(probes []StatusProbe) *Status {
	s.probes = probes
	return s
}

func (s *Status) Snapshot(ctx context.Context, actor identity.User) ([]port.ComponentStatus, error) {
	if actor.Role != identity.RolePlatformAdmin {
		return nil, identity.ErrNotFound
	}
	timeout := s.timeout()
	if timeout <= 0 {
		timeout = defaultStatusTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	items := make([]port.ComponentStatus, 0, len(s.probes))
	for _, probe := range s.probes {
		ok := true
		if probe.Ping != nil {
			ok = probe.Ping(ctx) == nil
		}
		items = append(items, port.ComponentStatus{ID: probe.ID, OK: ok})
	}
	return items, nil
}

func HTTPGetProbe(client *http.Client, rawURL string) func(context.Context) error {
	if client == nil {
		client = http.DefaultClient
	}
	return func(ctx context.Context) error {
		if strings.TrimSpace(rawURL) == "" {
			return errors.New("document health url is not configured")
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return err
		}
		response, err := client.Do(request)
		if err != nil {
			return err
		}
		defer func() { _, _ = io.Copy(io.Discard, response.Body); _ = response.Body.Close() }()
		if response.StatusCode >= 400 {
			return fmt.Errorf("health status %d", response.StatusCode)
		}
		return nil
	}
}
