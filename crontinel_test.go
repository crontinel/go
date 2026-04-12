package crontinel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientRequiresAPIKey(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			assert.Contains(t, r.(string), "api_key is required")
		} else {
			t.Fatal("expected panic")
		}
	}()
	NewClient("")
}

func TestNewClientWithOptions(t *testing.T) {
	c := NewClient("key", WithAPIURL("https://custom.example.com"), WithAppName("my-worker"))
	assert.Equal(t, "https://custom.example.com", c.apiURL)
	assert.Equal(t, "my-worker", c.appName)
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("key")
	assert.Equal(t, "https://app.crontinel.com", c.apiURL)
	assert.Equal(t, "go", c.appName)
}

func TestScheduleRunPayload(t *testing.T) {
	var captured map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": 1, "result": map[string]interface{}{"ok": true}})
	}))
	defer server.Close()

	c := NewClient("test_key", WithAPIURL(server.URL))
	err := c.ScheduleRun("php artisan schedule:run", 1500, 0)
	require.NoError(t, err)

	assert.Equal(t, "2.0", captured["jsonrpc"])
	assert.Equal(t, "notify/schedule_run", captured["method"])
	params := captured["params"].(map[string]interface{})
	assert.Equal(t, "php artisan schedule:run", params["command"])
	assert.Equal(t, float64(1500), params["duration_ms"])
	assert.Equal(t, float64(0), params["exit_code"])
	assert.Equal(t, "go", params["app"])
	assert.NotEmpty(t, params["ran_at"])
}

func TestScheduleRunWithRanAt(t *testing.T) {
	var captured map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": 1, "result": map[string]interface{}{"ok": true}})
	}))
	defer server.Close()

	c := NewClient("test_key", WithAPIURL(server.URL))
	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	err := c.ScheduleRun("test", 100, 0, ts)
	require.NoError(t, err)

	params := captured["params"].(map[string]interface{})
	assert.Equal(t, "2025-01-15T10:30:00Z", params["ran_at"])
}

func TestScheduleRunReportsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	c := NewClient("test_key", WithAPIURL(server.URL))
	err := c.ScheduleRun("test", 100, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestQueueProcessedPayload(t *testing.T) {
	var captured map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": 1, "result": map[string]interface{}{"ok": true}})
	}))
	defer server.Close()

	c := NewClient("test_key", WithAPIURL(server.URL))
	err := c.QueueProcessed("emails", 50, 2, 3200)
	require.NoError(t, err)

	params := captured["params"].(map[string]interface{})
	assert.Equal(t, "notify/queue_processed", captured["method"])
	assert.Equal(t, "emails", params["queue"])
	assert.Equal(t, float64(50), params["processed"])
	assert.Equal(t, float64(2), params["failed"])
	assert.Equal(t, float64(3200), params["duration_ms"])
}

func TestQueueProcessedDefaults(t *testing.T) {
	var captured map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": 1, "result": map[string]interface{}{"ok": true}})
	}))
	defer server.Close()

	c := NewClient("test_key", WithAPIURL(server.URL))
	err := c.QueueProcessed("default", 0, 0, 100)
	require.NoError(t, err)
	params := captured["params"].(map[string]interface{})
	assert.Equal(t, float64(0), params["processed"])
	assert.Equal(t, float64(0), params["failed"])
}

func TestHorizonSnapshotPayload(t *testing.T) {
	var captured map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": 1, "result": map[string]interface{}{"ok": true}})
	}))
	defer server.Close()

	c := NewClient("test_key", WithAPIURL(server.URL))
	supervisors := map[string]interface{}{
		"emails":  map[string]interface{}{"status": "running"},
		"reports": map[string]interface{}{"status": "paused"},
	}
	err := c.HorizonSnapshot(supervisors, 4.2, false)
	require.NoError(t, err)

	params := captured["params"].(map[string]interface{})
	assert.Equal(t, "notify/horizon_snapshot", captured["method"])
	sups := params["supervisors"].(map[string]interface{})
	assert.Equal(t, "running", sups["emails"].(map[string]interface{})["status"])
	assert.Equal(t, "paused", sups["reports"].(map[string]interface{})["status"])
	assert.Equal(t, 4.2, params["failed_jobs_per_minute"])
	assert.Equal(t, false, params["paused"])
}

func TestEventPayload(t *testing.T) {
	var captured map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": 1, "result": map[string]interface{}{"ok": true}})
	}))
	defer server.Close()

	c := NewClient("test_key", WithAPIURL(server.URL))
	metadata := map[string]interface{}{"version": "2.1.0"}
	err := c.Event("deployment", "Application deployed", "info", metadata)
	require.NoError(t, err)

	params := captured["params"].(map[string]interface{})
	assert.Equal(t, "notify/event", captured["method"])
	assert.Equal(t, "deployment", params["key"])
	assert.Equal(t, "Application deployed", params["message"])
	assert.Equal(t, "info", params["state"])
	meta := params["metadata"].(map[string]interface{})
	assert.Equal(t, "2.1.0", meta["version"])
}

func TestEventDefaultState(t *testing.T) {
	var captured map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": 1, "result": map[string]interface{}{"ok": true}})
	}))
	defer server.Close()

	c := NewClient("test_key", WithAPIURL(server.URL))
	err := c.Event("test", "Test event", "", map[string]interface{}{})
	require.NoError(t, err)
	params := captured["params"].(map[string]interface{})
	assert.Equal(t, "", params["state"])
}

func TestMonitorScheduleSuccess(t *testing.T) {
	var captured map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": 1, "result": map[string]interface{}{"ok": true}})
	}))
	defer server.Close()

	c := NewClient("test_key", WithAPIURL(server.URL))
	fn := func() error {
		time.Sleep(5 * time.Millisecond)
		return nil
	}
	ms, exitCode := c.MonitorSchedule("my-task", fn)

	assert.True(t, ms >= 5)
	assert.Equal(t, 0, exitCode)
	bodyParams := captured["params"].(map[string]interface{})
	assert.Equal(t, "my-task", bodyParams["command"])
	assert.Equal(t, float64(0), bodyParams["exit_code"])
}

func TestMonitorScheduleFailure(t *testing.T) {
	var captured map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": 1, "result": map[string]interface{}{"ok": true}})
	}))
	defer server.Close()

	c := NewClient("test_key", WithAPIURL(server.URL))
	fn := func() error {
		return fmt.Errorf("task failed")
	}
	_, exitCode := c.MonitorSchedule("failing-task", fn)
	assert.Equal(t, 1, exitCode)

	bodyParams := captured["params"].(map[string]interface{})
	assert.Equal(t, "failing-task", bodyParams["command"])
	assert.Equal(t, float64(1), bodyParams["exit_code"])
}

