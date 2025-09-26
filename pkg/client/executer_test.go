//go:build linux

package client

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amitschendel/curing/pkg/common"
	"github.com/amitschendel/curing/pkg/config"
	"github.com/amitschendel/curing/pkg/server"
	"github.com/stretchr/testify/assert"
)

func TestExecuterWithRealServer(t *testing.T) {
	// Configure logging for tests
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("Starting test")

	// Start server
	testPort := 8089
	testHost := "127.0.0.1"
	srv := server.NewServer(testHost, testPort)
	go srv.Run()

	// Give the server time to start
	time.Sleep(500 * time.Millisecond)

	// Create config
	cfg := &config.Config{
		ConnectIntervalSec: 10,
		Server: config.ServerDetails{
			Host: testHost,
			Port: testPort,
		},
	}

	parentCtx := context.Background()
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	// Create executer with 3 workers
	executer, err := NewExecuter(ctx, 3)
	assert.NoError(t, err)
	defer executer.Close()

	// Create command puller
	puller, err := NewCommandPuller(cfg, ctx, executer)
	assert.NoError(t, err)
	assert.NotNil(t, puller)
	defer puller.Close()

	// Channel to signal when we get a result
	resultReceived := make(chan struct{})
	resultReceivedOnce := sync.Once{}

	// Start components
	go executer.Run()
	go puller.Run()

	// Start a goroutine to monitor the output channel
	go func() {
		outputChan := executer.GetOutputChannel()
		for {
			select {
			case result, ok := <-outputChan:
				if !ok {
					return // channel closed
				}

				// Print limited output for logging
				outputPreview := ""
				if len(result.Output) > 0 {
					lines := strings.Split(string(result.Output), "\n")
					if len(lines) > 0 {
						outputPreview = lines[0]
						if len(lines) > 1 {
							outputPreview += "... (truncated)"
						}
					}
				}

				slog.Info("Result received",
					"commandID", result.CommandID,
					"returnCode", result.ReturnCode,
					"outputPreview", outputPreview,
					"outputSize", len(result.Output))

				// Signal that we got a result (only do this once)
				resultReceivedOnce.Do(func() {
					close(resultReceived)
				})
			case <-ctx.Done():
				return
			}
		}
	}()

	// Add a test command directly to ensure we get a result
	testCmd := common.ReadFile{
		Id:   "test-command",
		Path: "/etc/passwd", // This file should exist on most Linux systems
	}

	slog.Info("Sending test command directly to executer")
	select {
	case executer.commands <- testCmd:
		slog.Info("Test command sent")
	case <-time.After(1 * time.Second):
		t.Fatal("Failed to send test command to executer")
	}

	// Wait for the result with timeout
	select {
	case <-resultReceived:
		slog.Info("Test command executed successfully")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for test command result")
	}

	// Test passes if we reach here
	slog.Info("Test completed successfully")
}
