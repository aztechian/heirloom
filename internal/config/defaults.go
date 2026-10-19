package config

import (
	"github.com/spf13/viper"
)

const (
	DefaultPort             = "8080"
	DefaultReadTimeout      = 10
	DefaultWriteTimeout     = 10
	DefaultS3Region         = "us-east-1"
	DefaultS3Endpoint       = "https://s3.us-east-1.amazonaws.com"
	DefaultDataPath         = "./data"
	DefaultMaxAssetsBytes   = 100 * 1024 * 1024 // 100 MB
	DefaultAllowedMimeTypes = "image/jpeg,image/png,image/gif,image/webp,image/tiff,video/mp4,video/quicktime,video/x-matroska,video/x-msvideo"
)

func setDefaults() {
	viper.SetDefault("version", version) // set default version from build-time variable
	viper.SetDefault("server.log_level", "info")
	viper.SetDefault("server.port", DefaultPort)
	viper.SetDefault("server.read_timeout", DefaultReadTimeout)
	viper.SetDefault("server.write_timeout", DefaultWriteTimeout)
	viper.SetDefault("server.data_path", DefaultDataPath)
	viper.SetDefault("server.max_assets_bytes", DefaultMaxAssetsBytes)
	viper.SetDefault("s3.region", DefaultS3Region)
	viper.SetDefault("s3.endpoint", DefaultS3Endpoint)
	viper.SetDefault("server.allowed_mime_types", DefaultAllowedMimeTypes)
}

func bindEnvs() {
	viper.BindEnv("server.log_level", "LOG_LEVEL")
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("server.read_timeout", "SERVER_READ_TIMEOUT")
	viper.BindEnv("server.write_timeout", "SERVER_WRITE_TIMEOUT")
	viper.BindEnv("server.data_path", "SERVER_DATA_PATH")
	viper.BindEnv("server.max_assets_bytes", "SERVER_MAX_ASSETS_BYTES")
	viper.BindEnv("s3.region", "AWS_REGION")
	viper.BindEnv("s3.endpoint", "AWS_S3_ENDPOINT")
	viper.BindEnv("s3.access_key_id", "AWS_ACCESS_KEY_ID")
	viper.BindEnv("s3.secret_access_key", "AWS_SECRET_ACCESS_KEY")
	viper.BindEnv("server.allowed_mime_types", "SERVER_ALLOWED_MIME_TYPES")
}
