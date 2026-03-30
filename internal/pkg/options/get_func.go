package options

import (
	"fmt"
	"os"
	"time"
)

func GetDefaultConfigPath() string {
	return DefaultConfigPath
}

func getActiveConfig() *Config {
	if defaultConfig != nil {
		return defaultConfig
	}
	return builtinDefaults
}

func GetMainConf() *MainConf {
	cfg := getActiveConfig()
	if cfg != nil && cfg.Main != nil {
		return cfg.Main
	}
	return builtinDefaults.Main
}

func GetMainMonitoringPort() int {
	if cfg := GetMainConf(); cfg != nil {
		return cfg.MonitoringPort
	}
	return builtinDefaults.Main.MonitoringPort
}

// GetShutdownTimeout 返回 HTTP 服务优雅退出超时（用于 pkg/graceful）。
func GetShutdownTimeout() time.Duration {
	sec := 0
	if cfg := GetMainConf(); cfg != nil {
		sec = cfg.ShutdownTimeoutSeconds
	}
	if sec <= 0 {
		sec = builtinDefaults.Main.ShutdownTimeoutSeconds
	}
	if sec <= 0 {
		sec = 20
	}
	return time.Duration(sec) * time.Second
}

func GetLoggerConf() *LoggerConf {
	cfg := getActiveConfig()
	if cfg != nil && cfg.Logger != nil {
		return cfg.Logger
	}
	return builtinDefaults.Logger
}

func GetLoggerCompress() bool {
	return GetLoggerConf().Compress
}

func GetLoggerMaxSize() int {
	return GetLoggerConf().MaxSize
}

func GetLoggerMaxBackups() int {
	return GetLoggerConf().MaxBackups
}

func GetLoggerMaxAge() int {
	return GetLoggerConf().MaxAge
}

func GetLoggerLogDir() string {
	return GetLoggerConf().LogDir
}

func GetLoggerLogFileName() string {
	return GetLoggerConf().LogFileName
}

func GetLoggerLogLevel() string {
	return GetLoggerConf().LogLevel
}

func GetGatewayConf() *GatewayConf {
	cfg := getActiveConfig()
	if cfg != nil && cfg.Gateway != nil {
		return cfg.Gateway
	}
	return builtinDefaults.Gateway
}

func GetGatewayHTTPPort() int {
	return GetGatewayConf().HTTPPort
}

func GetGatewayGRPCPort() int {
	return GetGatewayConf().GRPCPort
}

func GetGatewayMonitoringPort() int {
	return GetGatewayConf().MonitoringPort
}

func GetGatewayLogFileName() string {
	return GetGatewayConf().LogFileName
}

func GetGatewayAuthBaseURL() string {
	return GetGatewayConf().AuthBaseURL
}

func GetGatewayConfigCenterBaseURL() string {
	return GetGatewayConf().ConfigCenterBaseURL
}

func GetEtcdConf() *EtcdConf {
	cfg := getActiveConfig()
	if cfg != nil && cfg.Etcd != nil {
		return cfg.Etcd
	}
	return nil
}

func GetAuthConf() *AuthConf {
	cfg := getActiveConfig()
	if cfg != nil && cfg.Auth != nil {
		return cfg.Auth
	}
	return builtinDefaults.Auth
}

func GetAuthHTTPPort() int {
	return GetAuthConf().HTTPPort
}

func GetAuthGRPCPort() int {
	return GetAuthConf().GRPCPort
}

func GetAuthMonitoringPort() int {
	return GetAuthConf().MonitoringPort
}

func GetAuthLogFileName() string {
	return GetAuthConf().LogFileName
}

// GetBootstrapCipherKey 解密 bootstrap 管理员密文的主密钥：优先环境变量。
func GetBootstrapCipherKey() string {
	if k := os.Getenv("OPSHUB_BOOTSTRAP_CIPHER_KEY"); k != "" {
		return k
	}
	if c := GetAuthConf(); c != nil {
		return c.BootstrapCipherKey
	}
	return ""
}

func GetConfigCenterConf() *ConfigCenterConf {
	cfg := getActiveConfig()
	if cfg != nil && cfg.ConfigCenter != nil {
		return cfg.ConfigCenter
	}
	return builtinDefaults.ConfigCenter
}

func GetConfigCenterHTTPPort() int {
	return GetConfigCenterConf().HTTPPort
}

func GetConfigCenterGRPCPort() int {
	return GetConfigCenterConf().GRPCPort
}

func GetConfigCenterMonitoringPort() int {
	return GetConfigCenterConf().MonitoringPort
}

func GetConfigCenterLogFileName() string {
	return GetConfigCenterConf().LogFileName
}

func GetMySQLConf() *MySQLConf {
	cfg := getActiveConfig()
	if cfg != nil && cfg.MySQL != nil {
		return cfg.MySQL
	}
	return builtinDefaults.MySQL
}

func GetMySQLPort() int {
	return GetMySQLConf().Port
}

func GetMySQLHost() string {
	return GetMySQLConf().Host
}

func GetMySQLUser() string {
	return GetMySQLConf().User
}

func GetMySQLPassword() string {
	return GetMySQLConf().Password
}

func GetMySQLDatabase() string {
	return GetMySQLConf().Database
}

func GetRedisConf() *RedisConf {
	cfg := getActiveConfig()
	if cfg != nil && cfg.Redis != nil {
		return cfg.Redis
	}
	return builtinDefaults.Redis
}

func GetRedisPort() int {
	return GetRedisConf().Port
}

func GetRedisDB() int {
	return GetRedisConf().DB
}

func GetRedisHost() string {
	return GetRedisConf().Host
}

func GetRedisPassword() string {
	return GetRedisConf().Password
}

func GetDefaultConfig() *Config {
	return getActiveConfig()
}

// GetMySQLOptionsFromConfig 从当前配置构建 MySQLOptions，供 IAM 等模块初始化 store 使用。
func GetMySQLOptionsFromConfig() *MySQLOptions {
	return &MySQLOptions{
		Host:                  fmt.Sprintf("%s:%d", GetMySQLHost(), GetMySQLPort()),
		Username:              GetMySQLUser(),
		Password:              GetMySQLPassword(),
		Database:              GetMySQLDatabase(),
		MaxIdleConnections:    100,
		MaxOpenConnections:    100,
		MaxConnectionLifeTime: NewMySQLOptions().MaxConnectionLifeTime,
		LogLevel:              NewMySQLOptions().LogLevel,
	}
}
