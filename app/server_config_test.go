package app

import (
	"testing"
	"time"
)

func TestServerConfigApplied(t *testing.T) {
	application := New()
	config := ServerConfig{
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       2 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       4 * time.Second,
		MaxHeaderBytes:    4096,
	}
	application.SetServerConfig(config)
	server, listener, err := application.prepareServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
		application.serverMu.Lock()
		if application.server == server {
			application.server = nil
		}
		application.serverMu.Unlock()
	})
	if server.ReadHeaderTimeout != config.ReadHeaderTimeout ||
		server.ReadTimeout != config.ReadTimeout ||
		server.WriteTimeout != config.WriteTimeout ||
		server.IdleTimeout != config.IdleTimeout ||
		server.MaxHeaderBytes != config.MaxHeaderBytes {
		t.Fatalf("server config = %#v, want %#v", server, config)
	}
}

func TestDefaultServerConfigIsBounded(t *testing.T) {
	config := DefaultServerConfig()
	if config.ReadHeaderTimeout <= 0 || config.ReadTimeout <= 0 ||
		config.WriteTimeout <= 0 || config.IdleTimeout <= 0 || config.MaxHeaderBytes <= 0 {
		t.Fatalf("unsafe defaults: %#v", config)
	}
}
