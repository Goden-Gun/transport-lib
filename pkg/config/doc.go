// Package config provides common configuration types and utilities
// for GGA family services (GGA, SideCar, GGA-Worker).
//
// Usage:
//
//	import "github.com/Goden-Gun/transport-lib/pkg/config"
//
//	type MyConfig struct {
//	    App   config.AppConfig   `yaml:"app" mapstructure:"app"`
//	    Redis config.RedisConfig `yaml:"redis" mapstructure:"redis"`
//	    Log   config.LogConfig   `yaml:"log" mapstructure:"log"`
//	    // ... service-specific configs
//	}
//
//	func LoadMyConfig() (*MyConfig, error) {
//	    cfg := &MyConfig{}
//	    if err := config.LoadConfig(cfg); err != nil {
//	        return nil, err
//	    }
//	    cfg.App.Env = config.GetEnv()
//	    return cfg, nil
//	}
package config
