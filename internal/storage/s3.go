package storage

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aztechian/heirloom/internal/config"
	"github.com/rs/zerolog"
)

// DefaultS3Timeout bounds the duration of any single S3 API call.
const DefaultS3Timeout = 30 * time.Second

type S3Storage struct {
	Logger *zerolog.Logger
	Client *s3.Client
	Bucket string
}

func NewS3Storage(ctx context.Context, cfg config.Config) *S3Storage {
	logger := zerolog.Ctx(ctx)
	s3config, err := loadS3Config(ctx, cfg.S3)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to load S3 config")
		return nil
	}
	client := s3.NewFromConfig(s3config)

	return &S3Storage{
		Logger: logger,
		Client: client,
		Bucket: cfg.S3.Bucket,
	}
}

func loadS3Config(ctx context.Context, cfg config.S3Config) (aws.Config, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}
	if cfg.Endpoint != "" {
		opts = append(opts, awsconfig.WithBaseEndpoint(cfg.Endpoint))
	}

	return awsconfig.LoadDefaultConfig(ctx, opts...)
}

func (s *S3Storage) Initialize(ctx context.Context, collection string) error {
	// Initialization logic for S3 storage goes here
	if err := s.createCollection(ctx, collection); err != nil {
		return err
	}
	// likely needs additional setup for upload directories, parquet files in the future
	return nil
}

func (s *S3Storage) createCollection(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultS3Timeout)
	defer cancel()

	// Logic to create a collection in S3 storage goes here
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String("/" + name + "/"),
		Body:   nil, // explicitly a nil body to create a "directory"
	}

	if exists, err := s.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: input.Bucket,
		Key:    input.Key,
	}); err == nil && exists != nil {
		// Collection already exists
		return nil
	}
	_, err := s.Client.PutObject(ctx, input)

	return err
}
