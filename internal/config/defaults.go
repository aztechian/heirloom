package config

import (
	"github.com/spf13/viper"
)

const (
	DefaultPort         = "8080"
	DefaultReadTimeout  = 10
	DefaultWriteTimeout = 10
	DefaultS3Region     = "us-east-1"
	DefaultS3Endpoint   = "https://s3.us-east-1.amazonaws.com"
)

func setDefaults() {
	viper.SetDefault("server.port", DefaultPort)
	viper.SetDefault("server.read_timeout", DefaultReadTimeout)
	viper.SetDefault("server.write_timeout", DefaultWriteTimeout)
	viper.SetDefault("s3.region", DefaultS3Region)
	viper.SetDefault("s3.endpoint", DefaultS3Endpoint)
}

func bindEnvs() {
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("server.read_timeout", "SERVER_READ_TIMEOUT")
	viper.BindEnv("server.write_timeout", "SERVER_WRITE_TIMEOUT")
	viper.BindEnv("s3.region", "AWS_REGION")
	viper.BindEnv("s3.endpoint", "AWS_S3_ENDPOINT")
	viper.BindEnv("s3.access_key_id", "AWS_ACCESS_KEY_ID")
	viper.BindEnv("s3.secret_access_key", "AWS_SECRET_ACCESS_KEY")
}
