package crontinel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client is the Crontinel API client.
type Client struct {
	apiKey  string
	apiURL  string
	appName string
	httpClient *http.Client
}

// Option is a functional option for NewClient.
type Option func(*Client)

// WithAPIURL sets a custom API URL.
func WithAPIURL(url string) Option {
	return func(c *Client) { c.apiURL = url }
}

// WithAppName sets a custom app name (default: "go").
func WithAppName(name string) Option {
	return func(c *Client) { c.appName = name }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// NewClient creates a new Crontinel client.
// It panics if apiKey is empty.
func NewClient(apiKey string, opts ...Option) *Client {
	if apiKey == "" {
		panic("crontinel: api_key is required")
	}
	c := &Client{
		apiKey:    apiKey,
		apiURL:    "https://app.crontinel.com",
		appName:   "go",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type jsonRPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Client) do(method string, params map[string]interface{}) error {
	params["app"] = c.appName
	reqBody, _ := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	})
	req, err := http.NewRequest(http.MethodPost, c.apiURL+"/api/mcp", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("crontinel: HTTP %d", resp.StatusCode)
	}

	var rpcResp jsonRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return err
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("crontinel: JSON-RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return nil
}

// ScheduleRun reports a scheduled command execution.
func (c *Client) ScheduleRun(command string, durationMs int, exitCode int, ranAt ...time.Time) error {
	params := map[string]interface{}{
		"command":     command,
		"duration_ms": durationMs,
		"exit_code":   exitCode,
	}
	if len(ranAt) > 0 {
		params["ran_at"] = ranAt[0].Format(time.RFC3339)
	} else {
		params["ran_at"] = time.Now().UTC().Format(time.RFC3339)
	}
	return c.do("notify/schedule_run", params)
}

// QueueProcessed reports queue worker activity.
func (c *Client) QueueProcessed(queue string, processed int, failed int, durationMs int, ranAt ...time.Time) error {
	params := map[string]interface{}{
		"queue":       queue,
		"processed":  processed,
		"failed":     failed,
		"duration_ms": durationMs,
	}
	if len(ranAt) > 0 {
		params["ran_at"] = ranAt[0].Format(time.RFC3339)
	} else {
		params["ran_at"] = time.Now().UTC().Format(time.RFC3339)
	}
	return c.do("notify/queue_processed", params)
}

// HorizonSnapshot reports Horizon supervisor status.
func (c *Client) HorizonSnapshot(supervisors map[string]interface{}, failedJobsPerMinute float64, paused bool, ranAt ...time.Time) error {
	params := map[string]interface{}{
		"supervisors":            supervisors,
		"failed_jobs_per_minute": failedJobsPerMinute,
		"paused":                  paused,
	}
	if len(ranAt) > 0 {
		params["ran_at"] = ranAt[0].Format(time.RFC3339)
	} else {
		params["ran_at"] = time.Now().UTC().Format(time.RFC3339)
	}
	return c.do("notify/horizon_snapshot", params)
}

// Event sends a custom event or alert.
func (c *Client) Event(key string, message string, state string, metadata map[string]interface{}, ranAt ...time.Time) error {
	params := map[string]interface{}{
		"key":       key,
		"message":  message,
		"state":    state,
		"metadata": metadata,
	}
	if len(ranAt) > 0 {
		params["ran_at"] = ranAt[0].Format(time.RFC3339)
	} else {
		params["ran_at"] = time.Now().UTC().Format(time.RFC3339)
	}
	return c.do("notify/event", params)
}

// MonitorSchedule runs fn and reports its outcome as a scheduled command.
// Returns (result, duration_ms, exit_code).
// If fn panics, exit_code is 1 and the panic is re-raised.
func (c *Client) MonitorSchedule(command string, fn func() error) (int64, int) {
	start := time.Now()
	exitCode := 0
	err := fn()
	if err != nil {
		exitCode = 1
	}
	durationMs := time.Since(start).Milliseconds()
	c.ScheduleRun(command, int(durationMs), exitCode)
	return durationMs, exitCode
}
