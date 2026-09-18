package app

import (
	"context"
	"testing"
	"time"
)

func TestRunUntilStopsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := New().runUntil(ctx, "127.0.0.1:0", time.Second); err != nil {
		t.Fatal(err)
	}
}

func TestShutdownWithoutActiveServer(t *testing.T) {
	if err := New().Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}
