# Crontinel Go Client

```go
package main

import (
    "time"
    "github.com/crontinel/go"
)

func main() {
    client := crontinel.NewClient("your_api_key")

    // Report a scheduled command
    client.ScheduleRun("php artisan schedule:run", 1500, 0)

    // Report queue processing
    client.QueueProcessed("emails", 50, 2, 3200)

    // Send a custom event
    client.Event("deployment", "Application deployed", "info", map[string]interface{}{"version": "2.0"})

    // Monitor a function
    durationMs, exitCode := client.MonitorSchedule("my-task", func() error {
        // do work
        return nil
    })
}
```

## Features

- `ScheduleRun` — report scheduled command outcome
- `QueueProcessed` — report queue worker activity
- `HorizonSnapshot` — report Laravel Horizon supervisor status
- `Event` — send custom events and alerts
- `MonitorSchedule` — run a function and auto-report outcome

## Options

```go
client := crontinel.NewClient("key",
    crontinel.WithAPIURL("https://custom.example.com"),
    crontinel.WithAppName("my-worker"),
)
```

## Laravel Integration

For Laravel applications, use the official [`crontinel/laravel`](https://github.com/crontinel/crontinel) package which integrates with the scheduler and queue worker out of the box.
