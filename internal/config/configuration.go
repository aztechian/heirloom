package config

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// VERSION is set at build time via -ldflags, so it must be a package-level
// variable rather than a const or a value threaded through the call stack.
//
//nolint:gochecknoglobals // set from a build-time -ldflags value
var VERSION = "dev"

type ServerConfig struct {
	Port         string `mapstructure:"port"`
	LogLevel     string `mapstructure:"log_level"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

type S3Config struct {
	Bucket          string `mapstructure:"bucket"`
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Endpoint        string `mapstructure:"endpoint"`
}

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	S3     S3Config     `mapstructure:"s3"`
}

func LoadConfig() (*Config, error) {
	var cfg Config
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	setDefaults()
	bindEnvs()

	pflag.Parse()
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return &cfg, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, continue with defaults and env vars
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
