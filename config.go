package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// Config 完整配置结构
type Config struct {
	Monitor   MonitorConfig   `json:"monitor"`
	RateLimit RateLimitConfig `json:"rate_limit"`
	Threshold LoadThreshold   `json:"threshold"`
	Server    ServerConfig    `json:"server"`
	Logging   LoggingConfig   `json:"logging"`
	Metrics   MetricsConfig   `json:"metrics"`
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	MonitorInterval string `json:"monitor_interval"` // 监控间隔
	AdjustInterval  string `json:"adjust_interval"`  // 调整间隔
	EnableCPU       bool   `json:"enable_cpu"`       // 启用CPU监控
	EnableMemory    bool   `json:"enable_memory"`    // 启用内存监控
	EnableRT        bool   `json:"enable_rt"`        // 启用响应时间监控
	EnableError     bool   `json:"enable_error"`     // 启用错误率监控
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         int    `json:"port"`
	Host         string `json:"host"`
	ReadTimeout  string `json:"read_timeout"`
	WriteTimeout string `json:"write_timeout"`
	EnableHTTP   bool   `json:"enable_http"`
	EnableGRPC   bool   `json:"enable_grpc"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `json:"level"`       // 日志级别
	Format     string `json:"format"`      // 日志格式
	Output     string `json:"output"`      // 输出目标
	MaxSize    int    `json:"max_size"`    // 最大文件大小(MB)
	MaxBackups int    `json:"max_backups"` // 最大备份数
	MaxAge     int    `json:"max_age"`     // 最大保存天数
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	EnablePrometheus bool   `json:"enable_prometheus"`
	PrometheusPort   int    `json:"prometheus_port"`
	MetricsPath      string `json:"metrics_path"`
	EnablePprof      bool   `json:"enable_pprof"`
	PprofPort        int    `json:"pprof_port"`
}

// ConfigManager 配置管理器
type ConfigManager struct {
	config     *Config
	configPath string
}

// NewConfigManager 创建配置管理器
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
	}
}

// LoadConfig 加载配置
func (cm *ConfigManager) LoadConfig() (*Config, error) {
	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		defaultConfig := cm.getDefaultConfig()
		if err := cm.SaveConfig(defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to save default config: %v", err)
		}
		cm.config = defaultConfig
		return defaultConfig, nil
	}

	// 读取配置文件
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	cm.config = &config
	return &config, nil
}

// SaveConfig 保存配置
func (cm *ConfigManager) SaveConfig(config *Config) error {
	// 确保目录存在
	dir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// 序列化配置
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// 写入文件
	if err := ioutil.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	cm.config = config
	return nil
}

// GetConfig 获取当前配置
func (cm *ConfigManager) GetConfig() *Config {
	return cm.config
}

// UpdateConfig 更新配置
func (cm *ConfigManager) UpdateConfig(updater func(*Config)) error {
	if cm.config == nil {
		return fmt.Errorf("config not loaded")
	}

	updater(cm.config)
	return cm.SaveConfig(cm.config)
}

// getDefaultConfig 获取默认配置
func (cm *ConfigManager) getDefaultConfig() *Config {
	return &Config{
		Monitor: MonitorConfig{
			MonitorInterval: "5s",
			AdjustInterval:  "10s",
			EnableCPU:       true,
			EnableMemory:    true,
			EnableRT:        true,
			EnableError:     true,
		},
		RateLimit: RateLimitConfig{
			BaseLimit:    1000,
			CurrentLimit: 1000,
			MinLimit:     100,
			MaxLimit:     5000,
			AdjustFactor: 1.0,
		},
		Threshold: LoadThreshold{
			CPUHigh:       70.0,
			CPUCritical:   85.0,
			MemHigh:       80.0,
			MemCritical:   90.0,
			RTHigh:        100.0,
			RTCritical:    300.0,
			ErrorHigh:     5.0,
			ErrorCritical: 10.0,
		},
		Server: ServerConfig{
			Port:         8080,
			Host:         "0.0.0.0",
			ReadTimeout:  "30s",
			WriteTimeout: "30s",
			EnableHTTP:   true,
			EnableGRPC:   false,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
		},
		Metrics: MetricsConfig{
			EnablePrometheus: true,
			PrometheusPort:   9090,
			MetricsPath:      "/metrics",
			EnablePprof:      true,
			PprofPort:        6060,
		},
	}
}

// ParseDuration 解析时间间隔字符串
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// ValidateConfig 验证配置
func (cm *ConfigManager) ValidateConfig(config *Config) error {
	// 验证监控间隔
	if _, err := ParseDuration(config.Monitor.MonitorInterval); err != nil {
		return fmt.Errorf("invalid monitor interval: %v", err)
	}

	if _, err := ParseDuration(config.Monitor.AdjustInterval); err != nil {
		return fmt.Errorf("invalid adjust interval: %v", err)
	}

	// 验证限流配置
	if config.RateLimit.MinLimit <= 0 {
		return fmt.Errorf("min limit must be positive")
	}

	if config.RateLimit.MaxLimit <= config.RateLimit.MinLimit {
		return fmt.Errorf("max limit must be greater than min limit")
	}

	if config.RateLimit.BaseLimit < config.RateLimit.MinLimit ||
		config.RateLimit.BaseLimit > config.RateLimit.MaxLimit {
		return fmt.Errorf("base limit must be between min and max limits")
	}

	// 验证阈值配置
	if config.Threshold.CPUHigh >= config.Threshold.CPUCritical {
		return fmt.Errorf("CPU high threshold must be less than critical threshold")
	}

	if config.Threshold.MemHigh >= config.Threshold.MemCritical {
		return fmt.Errorf("memory high threshold must be less than critical threshold")
	}

	// 验证服务器配置
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port")
	}

	if _, err := ParseDuration(config.Server.ReadTimeout); err != nil {
		return fmt.Errorf("invalid read timeout: %v", err)
	}

	if _, err := ParseDuration(config.Server.WriteTimeout); err != nil {
		return fmt.Errorf("invalid write timeout: %v", err)
	}

	return nil
}

// ReloadConfig 重新加载配置
func (cm *ConfigManager) ReloadConfig() error {
	config, err := cm.LoadConfig()
	if err != nil {
		return err
	}

	return cm.ValidateConfig(config)
}

// GetConfigPath 获取配置文件路径
func (cm *ConfigManager) GetConfigPath() string {
	return cm.configPath
}

// BackupConfig 备份配置文件
func (cm *ConfigManager) BackupConfig() error {
	if cm.config == nil {
		return fmt.Errorf("no config to backup")
	}

	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.backup_%s", cm.configPath, timestamp)

	return cm.SaveConfigToPath(cm.config, backupPath)
}

// SaveConfigToPath 保存配置到指定路径
func (cm *ConfigManager) SaveConfigToPath(config *Config, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	return ioutil.WriteFile(path, data, 0644)
}
