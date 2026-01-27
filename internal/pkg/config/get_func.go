package config

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
