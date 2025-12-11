package config

import "time"

// Duration 支持 YAML/JSON 反序列化，单位为秒
// 可以从数字（秒数）或字符串（如 "30s"）解析
type Duration int64

// Duration 返回 time.Duration 值
func (d Duration) Duration() time.Duration {
	return time.Duration(d) * time.Second
}

// Seconds 返回秒数
func (d Duration) Seconds() int64 {
	return int64(d)
}

// SecondsInt 返回 int 类型的秒数
func (d Duration) SecondsInt() int {
	return int(d)
}
