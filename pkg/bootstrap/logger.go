package bootstrap

import (
	"io"
	"os"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"

	"github.com/Goden-Gun/transport-lib/pkg/config"
)

// LogFileConfig 日志文件配置
type LogFileConfig struct {
	Enabled      bool   `yaml:"enabled" mapstructure:"enabled"`
	Dir          string `yaml:"dir" mapstructure:"dir"`
	Filename     string `yaml:"filename" mapstructure:"filename"`
	MaxAgeDays   int    `yaml:"max_age_days" mapstructure:"max_age_days"`
	RotationDays int    `yaml:"rotation_days" mapstructure:"rotation_days"`
}

// LoggerOptions 日志初始化选项
type LoggerOptions struct {
	// ServiceName 服务名称，用于日志文件命名
	ServiceName string
	// FileConfig 日志文件配置，nil 则不输出到文件
	FileConfig *LogFileConfig
	// AddContainerHook 是否添加容器ID钩子
	AddContainerHook bool
}

// containerHook 添加容器ID到日志
type containerHook struct {
	containerID string
}

func (h *containerHook) Levels() []log.Level {
	return log.AllLevels
}

func (h *containerHook) Fire(entry *log.Entry) error {
	entry.Data["container_id"] = h.containerID
	return nil
}

// detectContainerID 检测容器ID
func detectContainerID() string {
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		return hostname
	}

	if data, err := os.ReadFile("/etc/hostname"); err == nil {
		hostname := strings.TrimSpace(string(data))
		if hostname != "" {
			return hostname
		}
	}

	return "unknown"
}

// InitLogger 初始化日志，仅设置格式和级别，不输出到文件
func InitLogger(cfg config.LogConfig) error {
	return InitLoggerWithOptions(cfg, LoggerOptions{})
}

// InitLoggerWithFile 初始化日志并输出到文件
func InitLoggerWithFile(cfg config.LogConfig, serviceName string) error {
	return InitLoggerWithOptions(cfg, LoggerOptions{
		ServiceName: serviceName,
		FileConfig: &LogFileConfig{
			Enabled:      true,
			Dir:          "./logs",
			Filename:     serviceName,
			MaxAgeDays:   7,
			RotationDays: 1,
		},
		AddContainerHook: true,
	})
}

// InitLoggerWithOptions 使用完整选项初始化日志
func InitLoggerWithOptions(cfg config.LogConfig, opts LoggerOptions) error {
	// 设置日志格式
	switch cfg.Format {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	case "text":
		log.SetFormatter(&log.TextFormatter{})
	default:
		log.SetFormatter(&log.JSONFormatter{})
	}

	// 设置日志级别
	if lvl, err := log.ParseLevel(cfg.Level); err == nil {
		log.SetLevel(lvl)
	} else {
		log.SetLevel(log.InfoLevel)
		log.Warnf("invalid log level %q, fallback to info", cfg.Level)
	}

	// 设置打印调用信息
	log.SetReportCaller(cfg.ReportCaller)

	// 设置文件输出
	if opts.FileConfig != nil && opts.FileConfig.Enabled {
		if err := setupFileOutput(opts.FileConfig, opts.ServiceName); err != nil {
			return err
		}
	}

	// 添加容器钩子
	if opts.AddContainerHook {
		log.AddHook(&containerHook{containerID: detectContainerID()})
	}

	return nil
}

// setupFileOutput 设置日志文件输出
func setupFileOutput(fileCfg *LogFileConfig, serviceName string) error {
	logDir := fileCfg.Dir
	if logDir == "" {
		logDir = "./logs"
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Errorf("创建日志目录失败: %v", err)
		return err
	}

	filename := fileCfg.Filename
	if filename == "" {
		filename = serviceName
	}
	if filename == "" {
		filename = "app"
	}

	maxAge := fileCfg.MaxAgeDays
	if maxAge <= 0 {
		maxAge = 7
	}

	rotationDays := fileCfg.RotationDays
	if rotationDays <= 0 {
		rotationDays = 1
	}

	logFilePath := logDir + "/" + filename + ".%Y%m%d.log"
	linkName := logDir + "/" + filename + ".log"

	writer, err := rotatelogs.New(
		logFilePath,
		rotatelogs.WithLinkName(linkName),
		rotatelogs.WithMaxAge(time.Duration(maxAge)*24*time.Hour),
		rotatelogs.WithRotationTime(time.Duration(rotationDays)*24*time.Hour),
	)
	if err != nil {
		log.Errorf("设置日志输出失败: %v", err)
		return err
	}

	multiWriter := io.MultiWriter(os.Stdout, writer)
	log.SetOutput(multiWriter)

	return nil
}
