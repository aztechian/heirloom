package config

import "github.com/spf13/viper"

const (
	DefaultPort         = "8080"
	DefaultReadTimeout  = 10
	DefaultWriteTimeout = 10
)

func setDefaults() {
	viper.SetDefault("server.port", DefaultPort)
	viper.SetDefault("server.read_timeout", DefaultReadTimeout)
	viper.SetDefault("server.write_timeout", DefaultWriteTimeout)
}

func bindEnvs() {
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("server.read_timeout", "SERVER_READ_TIMEOUT")
	viper.BindEnv("server.write_timeout", "SERVER_WRITE_TIMEOUT")
}
