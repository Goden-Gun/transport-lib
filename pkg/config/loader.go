package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// LoadOptions 加载配置选项
type LoadOptions struct {
	ConfigPath    string // 配置文件目录，默认 "./configs"
	EnvPrefix     string // 环境变量前缀，用于 viper.AutomaticEnv
	AllowNoConfig bool   // 允许没有配置文件，纯环境变量配置
}

// LoadConfig 通用配置加载函数
// cfg 必须是指向配置结构体的指针
func LoadConfig(cfg interface{}, opts ...LoadOptions) error {
	opt := LoadOptions{ConfigPath: "./configs"}
	if len(opts) > 0 {
		opt = opts[0]
	}

	// 加载 .env 文件
	envFile := os.Getenv("ENV_FILE")
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("load %s failed: %w", envFile, err)
			}
		}
	} else {
		if err := godotenv.Load(); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("load .env failed: %w", err)
			}
		}
	}

	// 读取环境
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	viper.SetConfigName(fmt.Sprintf("config_%s", env))
	viper.SetConfigType("yaml")
	viper.AddConfigPath(opt.ConfigPath)

	// 配置环境变量支持
	if opt.EnvPrefix != "" {
		viper.SetEnvPrefix(opt.EnvPrefix)
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		viper.AutomaticEnv()
	}

	// 尝试读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) && opt.AllowNoConfig {
			// 允许没有配置文件，使用纯环境变量
		} else {
			return fmt.Errorf("read config failed: %w", err)
		}
	}

	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("unmarshal config failed: %w", err)
	}

	return nil
}

// GetEnv 获取当前环境，默认为 "dev"
func GetEnv() string {
	env := os.Getenv("APP_ENV")
	if env == "" {
		return "dev"
	}
	return env
}

// GetNodeID 获取节点 ID，按顺序尝试多个环境变量
// 如果都为空则返回空字符串，调用方可自行生成 UUID
func GetNodeID(envKeys ...string) string {
	for _, key := range envKeys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	if v := os.Getenv("HOSTNAME"); v != "" {
		return v
	}
	return ""
}
