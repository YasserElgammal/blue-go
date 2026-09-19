package app

import "time"

// ServerConfig controls the resource limits of the net/http server created by
// Start and Run. A zero timeout disables that timeout. A zero MaxHeaderBytes
// uses net/http's default limit.
type ServerConfig struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
}

// DefaultServerConfig returns conservative defaults for a REST API server.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

func validateServerConfig(config ServerConfig) {
	if config.ReadHeaderTimeout < 0 || config.ReadTimeout < 0 ||
		config.WriteTimeout < 0 || config.IdleTimeout < 0 {
		panic("blue: server timeouts cannot be negative")
	}
	if config.MaxHeaderBytes < 0 {
		panic("blue: MaxHeaderBytes cannot be negative")
	}
}
