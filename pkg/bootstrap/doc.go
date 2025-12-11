// Package bootstrap provides common initialization utilities for microservices.
//
// This package consolidates repeated initialization logic across services including:
//   - Logger setup with file rotation
//   - Redis connection management
//   - OpenTelemetry tracing initialization
//
// Example usage:
//
//	func main() {
//	    cfg := &config.Config{}
//	    if err := transportconfig.LoadConfig(cfg); err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Initialize logger
//	    if err := bootstrap.InitLogger(cfg.Log, "my-service"); err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Initialize Redis
//	    redisClient, err := bootstrap.InitRedis(ctx, cfg.Redis)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Initialize tracing
//	    shutdown, err := bootstrap.InitTracing(ctx, cfg.Tracing)
//	    if err != nil {
//	        log.Warn(err)
//	    }
//	    defer shutdown(ctx)
//	}
package bootstrap
