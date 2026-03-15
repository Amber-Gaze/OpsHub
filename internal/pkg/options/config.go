package options

import (
	"sync"

	"github.com/spf13/viper"
)

var (
	DefaultConfigPath = "../conf/ops_hub.yaml"
	defaultConfig     *Config
	once              sync.Once
)

type LoggerConf struct {
	Compress    bool   `mapstructure:"compress"`      // Whether to compress rotated log files
	MaxSize     int    `mapstructure:"max_size"`      // Maximum size in megabytes of the log file before it gets rotated
	MaxBackups  int    `mapstructure:"max_backups"`   // Maximum number of old log files to retain
	MaxAge      int    `mapstructure:"max_age"`       // Maximum number of days to retain old log files
	LogDir      string `mapstructure:"log_dir"`       // Directory to store log files
	LogFileName string `mapstructure:"log_file_name"` // Base name of the log file
	LogLevel    string `mapstructure:"log_level"`     // Minimum log level
}

type MainConf struct {
	MonitoringPort int `mapstructure:"monitoring_port"`
}

type GatewayConf struct {
	HTTPPort             int    `mapstructure:"http_port"`
	GRPCPort             int    `mapstructure:"grpc_port"`
	MonitoringPort       int    `mapstructure:"monitoring_port"`
	LogFileName          string `mapstructure:"log_file_name"`
	AuthBaseURL          string `mapstructure:"auth_base_url"`           // 可选，不填则用 http://127.0.0.1:{auth.http_port}
	ConfigCenterBaseURL  string `mapstructure:"config_center_base_url"` // 可选，不填则用 http://127.0.0.1:{config_center.http_port}
}

type AuthConf struct {
	HTTPPort       int    `mapstructure:"http_port"`
	GRPCPort       int    `mapstructure:"grpc_port"`
	MonitoringPort int    `mapstructure:"monitoring_port"`
	LogFileName    string `mapstructure:"log_file_name"`
}

type ConfigCenterConf struct {
	HTTPPort       int    `mapstructure:"http_port"`
	GRPCPort       int    `mapstructure:"grpc_port"`
	MonitoringPort int    `mapstructure:"monitoring_port"`
	LogFileName    string `mapstructure:"log_file_name"`
}

type MySQLConf struct {
	Port     int    `mapstructure:"port"`
	Host     string `mapstructure:"host"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}

type RedisConf struct {
	Port     int    `mapstructure:"port"`
	DB       int    `mapstructure:"db"`
	Host     string `mapstructure:"host"`
	Password string `mapstructure:"password"`
}

// EtcdConf 可选，用于配置中心 etcd 存储与发现（后续可接 ConfigMeta / ServiceRegistry）。
type EtcdConf struct {
	Endpoints []string `mapstructure:"endpoints"`
	Prefix    string   `mapstructure:"prefix"` // 配置 key 前缀，如 /opshub/config
}

type Config struct {
	Main         *MainConf         `mapstructure:"main"`
	Logger       *LoggerConf       `mapstructure:"logger"`
	MySQL        *MySQLConf        `mapstructure:"mysql"`
	Redis        *RedisConf        `mapstructure:"redis"`
	Etcd         *EtcdConf         `mapstructure:"etcd"`
	Auth         *AuthConf         `mapstructure:"auth"`
	ConfigCenter *ConfigCenterConf `mapstructure:"config_center"`
	Gateway      *GatewayConf      `mapstructure:"gateway"`
}

// LoadConfig reads the configuration from the specified YAML file and populates the Config struct.
func LoadConfig(configPath string) (*Config, error) {
	once.Do(func() {
		if configPath == "" {
			configPath = DefaultConfigPath
		}
		viper.SetConfigFile(configPath)
		viper.SetConfigType("yaml")

		// Read the configuration file
		if err := viper.ReadInConfig(); err != nil {
			return
		}

		// Unmarshal the configuration into the Config struct
		var cfg Config
		if err := viper.Unmarshal(&cfg); err != nil {
			return
		}

		defaultConfig = &cfg
	})

	if defaultConfig == nil {
		return nil, viper.ConfigFileNotFoundError{}
	}
	return defaultConfig, nil
}
