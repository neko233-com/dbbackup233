package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Reporter struct {
	cfg      ReportConfig
	identity IdentityConfig
	client   *http.Client
}

func NewReporter(cfg ReportConfig, identity ...IdentityConfig) Reporter {
	timeout := 5 * time.Second
	if cfg.Timeout != "" {
		if parsed, err := time.ParseDuration(cfg.Timeout); err == nil {
			timeout = parsed
		}
	}
	id := IdentityConfig{}
	if len(identity) > 0 {
		id = identity[0]
	}
	return Reporter{cfg: cfg, identity: id, client: &http.Client{Timeout: timeout}}
}

func (r Reporter) Enabled() bool {
	return r.cfg.Enabled && r.cfg.URL != ""
}

func (r Reporter) Send(ctx context.Context, event ReportEvent) error {
	if !r.Enabled() {
		return nil
	}
	event.Project = r.identity.Project
	event.Cluster = r.identity.Cluster
	event.MachineID = r.identity.MachineID
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	attempts := normalizeAttempts(r.cfg.Retry.Attempts)
	backoff := parseBackoff(r.cfg.Retry.Backoff)
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.cfg.URL, bytes.NewReader(raw))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		if r.cfg.Token != "" {
			req.Header.Set("Authorization", "Bearer "+r.cfg.Token)
		}
		for key, value := range r.cfg.Headers {
			req.Header.Set(key, value)
		}
		resp, err := r.client.Do(req)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			_ = resp.Body.Close()
			return nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("report endpoint returned %s", resp.Status)
			_ = resp.Body.Close()
		}
		if attempt < attempts {
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return lastErr
}

func normalizeAttempts(value int) int {
	if value <= 0 {
		return 1
	}
	return value
}

func parseBackoff(value string) time.Duration {
	if value == "" {
		return 200 * time.Millisecond
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 200 * time.Millisecond
	}
	return parsed
}
