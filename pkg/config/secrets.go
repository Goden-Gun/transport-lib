package config

import (
	"os"
	"strings"
)

// GetSecretOrEnv 从 Docker Secret 文件或环境变量读取敏感信息
// 优先级: {NAME}_FILE 指定的文件 > {NAME} 环境变量 > 默认值
//
// 示例:
//
//	password := GetSecretOrEnv("DB_PASSWORD", "default")
//	// 如果 DB_PASSWORD_FILE=/run/secrets/db-password 存在，读取文件内容
//	// 否则读取 DB_PASSWORD 环境变量
//	// 都不存在则返回 "default"
func GetSecretOrEnv(name string, defaultValue string) string {
	// 检查 {NAME}_FILE 环境变量
	filePath := os.Getenv(name + "_FILE")
	if filePath != "" {
		if data, err := os.ReadFile(filePath); err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	// 回退到环境变量
	if value := os.Getenv(name); value != "" {
		return value
	}

	return defaultValue
}

// MustGetSecret 从 Docker Secret 文件或环境变量读取敏感信息
// 如果找不到则 panic
func MustGetSecret(name string) string {
	value := GetSecretOrEnv(name, "")
	if value == "" {
		panic("required secret not found: " + name)
	}
	return value
}

// SecretLoader 批量加载 Secrets 到配置结构
type SecretLoader struct {
	secrets map[string]string
}

// NewSecretLoader 创建 Secret 加载器
func NewSecretLoader() *SecretLoader {
	return &SecretLoader{
		secrets: make(map[string]string),
	}
}

// Load 加载单个 Secret
func (l *SecretLoader) Load(name string, defaultValue string) *SecretLoader {
	l.secrets[name] = GetSecretOrEnv(name, defaultValue)
	return l
}

// MustLoad 加载必需的 Secret，找不到则 panic
func (l *SecretLoader) MustLoad(name string) *SecretLoader {
	l.secrets[name] = MustGetSecret(name)
	return l
}

// Get 获取已加载的 Secret
func (l *SecretLoader) Get(name string) string {
	return l.secrets[name]
}

// All 获取所有已加载的 Secrets
func (l *SecretLoader) All() map[string]string {
	result := make(map[string]string, len(l.secrets))
	for k, v := range l.secrets {
		result[k] = v
	}
	return result
}

// ApplyToConfig 将 Secrets 应用到配置结构
// 使用函数回调方式，避免反射开销
//
// 示例:
//
//	loader := NewSecretLoader().
//	    Load("DB_PASSWORD", "").
//	    Load("REDIS_PASSWORD", "").
//	    Load("JWT_SECRET", "")
//
//	loader.ApplyToConfig(func(secrets map[string]string) {
//	    cfg.Database.Password = secrets["DB_PASSWORD"]
//	    cfg.Redis.Password = secrets["REDIS_PASSWORD"]
//	    cfg.JWT.SecretKey = secrets["JWT_SECRET"]
//	})
func (l *SecretLoader) ApplyToConfig(applier func(secrets map[string]string)) {
	applier(l.secrets)
}

// LoadConfigWithSecrets 加载配置并注入 Secrets
// 这是 LoadConfig 的增强版本，支持 Docker Secrets
//
// 示例:
//
//	cfg := &Config{}
//	secretDefs := []SecretDefinition{
//	    {Name: "DB_PASSWORD", Target: &cfg.Database.Password},
//	    {Name: "REDIS_PASSWORD", Target: &cfg.Redis.Password, Default: ""},
//	    {Name: "JWT_SECRET", Target: &cfg.JWT.SecretKey, Required: true},
//	}
//	if err := LoadConfigWithSecrets(cfg, secretDefs); err != nil {
//	    log.Fatal(err)
//	}
func LoadConfigWithSecrets(cfg interface{}, secrets []SecretDefinition, opts ...LoadOptions) error {
	// 先加载 YAML 配置
	if err := LoadConfig(cfg, opts...); err != nil {
		return err
	}

	// 然后注入 Secrets
	for _, s := range secrets {
		value := GetSecretOrEnv(s.Name, s.Default)
		if s.Required && value == "" {
			return &SecretNotFoundError{Name: s.Name}
		}
		if s.Target != nil {
			*s.Target = value
		}
	}

	return nil
}

// SecretDefinition Secret 定义
type SecretDefinition struct {
	Name     string  // Secret 名称 (如 DB_PASSWORD)
	Target   *string // 目标字段指针
	Default  string  // 默认值
	Required bool    // 是否必需
}

// SecretNotFoundError Secret 未找到错误
type SecretNotFoundError struct {
	Name string
}

func (e *SecretNotFoundError) Error() string {
	return "required secret not found: " + e.Name
}
