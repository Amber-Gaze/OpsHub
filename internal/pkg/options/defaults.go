package options

var builtinDefaults = &Config{
	Main: &MainConf{
		MonitoringPort:         8100,
		ShutdownTimeoutSeconds: 20,
	},
	Logger: &LoggerConf{
		Compress:    false,
		MaxSize:     10,
		MaxBackups:  5,
		MaxAge:      7,
		LogDir:      "./logs",
		LogFileName: "ops_hub.log",
		LogLevel:    "info",
	},
	Gateway: &GatewayConf{
		HTTPPort:       8001,
		GRPCPort:       8002,
		MonitoringPort: 8003,
		LogFileName:    "ops_hub_gateway.log",
	},
	Auth: &AuthConf{
		HTTPPort:       8004,
		GRPCPort:       8005,
		MonitoringPort: 8006,
		LogFileName:    "ops_hub_auth.log",
	},
	ConfigCenter: &ConfigCenterConf{
		HTTPPort:       8007,
		GRPCPort:       8008,
		MonitoringPort: 8009,
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
