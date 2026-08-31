package options

var builtinDefaults = &Config{
	Main: &MainConf{
		MonitoringPort:         8100,
		ShutdownTimeoutSeconds: 20,
	},
	Logger: &LoggerConf{
		Compress:    false,
		MaxBackups:  30,
		MaxAge:      7,
		LogDir:      "../logs",
		LogFileName: "ops_hub.log",
		LogLevel:    "info",
		Rotation:    "hour",
		Encoding:    "json",
	},
	Gateway: &GatewayConf{
		HTTPPort:       8001,
		GRPCPort:       8002,
		MonitoringPort: 8003,
		LogFileName:    "ops_hub_gateway.log",
	},
	Auth: &AuthConf{
		HTTPPort:       8101,
		GRPCPort:       8102,
		MonitoringPort: 8103,
		LogFileName:    "ops_hub_auth.log",
	},
	ConfigCenter: &ConfigCenterConf{
		HTTPPort:       8201,
		GRPCPort:       8202,
		MonitoringPort: 8203,
		LogFileName:    "ops_hub_config.log",
	},
	MySQL: &MySQLConf{
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "liya1234",
		Database: "opshub_db",
	},
	Redis: &RedisConf{
		Host:     "localhost",
		Port:     6379,
		Password: "liya1234",
		DB:       0,
	},
}
