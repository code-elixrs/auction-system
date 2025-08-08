package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Redis    RedisConfig    `mapstructure:"redis"`
	MySQL    MySQLConfig    `mapstructure:"mysql"`
	Leader   LeaderConfig   `mapstructure:"leader"`
	Instance InstanceConfig `mapstructure:"instance"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type RedisConfig struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type MySQLConfig struct {
	DSN             string        `mapstructure:"dsn"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type LeaderConfig struct {
	TTL time.Duration `mapstructure:"ttl"`
}

type InstanceConfig struct {
	ID string `mapstructure:"id"`
}

func Load() (*Config, error) {
	// Set default values
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("redis.address", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("mysql.dsn", "auction_user:auction_pass@tcp(localhost:3306)/auction_db?parseTime=true")
	viper.SetDefault("mysql.max_open_conns", 25)
	viper.SetDefault("mysql.max_idle_conns", 10)
	viper.SetDefault("mysql.conn_max_lifetime", 5*time.Minute)
	viper.SetDefault("leader.ttl", 30*time.Second)
	viper.SetDefault("instance.id", "auction-service-1")

	// Configuration file settings
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/auction-system/")

	// Environment variable support
	viper.AutomaticEnv()

	// Environment variable mappings
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("server.host", "SERVER_HOST")
	viper.BindEnv("redis.address", "REDIS_ADDRESS")
	viper.BindEnv("redis.password", "REDIS_PASSWORD")
	viper.BindEnv("redis.db", "REDIS_DB")
	viper.BindEnv("mysql.dsn", "MYSQL_DSN")
	viper.BindEnv("mysql.max_open_conns", "MYSQL_MAX_OPEN_CONNS")
	viper.BindEnv("mysql.max_idle_conns", "MYSQL_MAX_IDLE_CONNS")
	viper.BindEnv("mysql.conn_max_lifetime", "MYSQL_CONN_MAX_LIFETIME")
	viper.BindEnv("leader.ttl", "LEADER_TTL")
	viper.BindEnv("instance.id", "INSTANCE_ID")

	// Read configuration file (optional - will use defaults/env vars if not found)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		// Config file not found, continue with defaults and environment variables
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// LoadFromFile loads configuration from a specific file path
func LoadFromFile(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetConfigString returns a formatted string representation of the config
func (c *Config) GetConfigString() string {
	return fmt.Sprintf(
		"Server: %s:%d, Redis: %s, MySQL: %s, Instance: %s",
		c.Server.Host,
		c.Server.Port,
		c.Redis.Address,
		c.MySQL.DSN,
		c.Instance.ID,
	)
}
