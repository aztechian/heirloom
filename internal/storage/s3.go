package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aztechian/heirloom/internal/api/types"
	"github.com/aztechian/heirloom/internal/config"
	"github.com/rs/zerolog"
)

// DefaultS3Timeout bounds the duration of any single S3 API call.
const DefaultS3Timeout = 30 * time.Second
const DefaultCollectionTags = "resource=collection"
const DefaultMetadataTags = "resource=metadata"

type S3Storage struct {
	Logger       *zerolog.Logger
	Client       *s3.Client
	Bucket       string
	MetadataPath string
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
		Logger:       logger,
		Client:       client,
		Bucket:       cfg.S3.Bucket,
		MetadataPath: filepath.Join("/", DefaultMetadataDir),
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

func (s *S3Storage) CollectionExists(ctx context.Context, name string) bool {
	ctx, cancel := context.WithTimeout(ctx, DefaultS3Timeout)
	defer cancel()

	_, err := s.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String("/" + name + "/"),
	})

	return err == nil
}

func (s *S3Storage) DeleteCollection(ctx context.Context, collection types.Collection) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultS3Timeout)
	defer cancel()

	_, err := s.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String("/" + collection.Slug + "/"),
	})
	if err != nil {
		s.Logger.Warn().Err(err).Str("collection", collection.Slug).Msg("Failed to delete collection")
		return err
	}

	return s.deleteCollectionMetadata(ctx, collection.Slug)
}

func (s *S3Storage) CreateCollection(ctx context.Context, collection types.Collection) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultS3Timeout)
	defer cancel()

	// Logic to create a collection in S3 storage goes here
	input := &s3.PutObjectInput{
		Bucket:  aws.String(s.Bucket),
		Key:     aws.String("/" + collection.Slug + "/"),
		Body:    nil, // explicitly a nil body to create a "directory"
		Tagging: aws.String("resource=collection"),
	}

	if exists, err := s.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: input.Bucket,
		Key:    input.Key,
	}); err == nil && exists != nil {
		// Collection already exists
		return nil
	}
	if _, err := s.Client.PutObject(ctx, input); err != nil {
		return err
	}

	return s.writeCollectionMetadata(ctx, collection) // ignoring error for now
}

func (s *S3Storage) ListCollections(ctx context.Context) []types.Collection {
	ctx, cancel := context.WithTimeout(ctx, DefaultS3Timeout)
	defer cancel()

	output, err := s.Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.Bucket),
		Prefix:    aws.String("/"),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil
	}

	collections := make([]types.Collection, 0, len(output.CommonPrefixes))
	for _, prefix := range output.CommonPrefixes {
		if prefix.Prefix != nil {
			if *prefix.Prefix == s.MetadataPath+"/" {
				continue
			}
			if collection, err := s.readCollectionMetadata(ctx, strings.Trim(*prefix.Prefix, "/")); err == nil {
				collections = append(collections, collection)
			}
		}
	}

	return collections
}

func (s *S3Storage) writeCollectionMetadata(ctx context.Context, collection types.Collection) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultS3Timeout)
	defer cancel()

	path := filepath.Join(s.MetadataPath, collection.Slug+".json")
	tags := strings.Join([]string{DefaultCollectionTags, DefaultMetadataTags}, ",")
	collectionJson, err := json.Marshal(collection)
	if err != nil {
		return err
	}

	input := &s3.PutObjectInput{
		Bucket:  aws.String(s.Bucket),
		Key:     aws.String(path),
		Body:    bytes.NewReader(collectionJson),
		Tagging: aws.String(tags),
	}
	_, err = s.Client.PutObject(ctx, input)

	return err
}

func (s *S3Storage) readCollectionMetadata(ctx context.Context, slug string) (types.Collection, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultS3Timeout)
	defer cancel()

	path := filepath.Join(s.MetadataPath, slug+".json")
	output, err := s.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return types.Collection{}, err
	}
	defer func() { _ = output.Body.Close() }()
	object, err := io.ReadAll(output.Body)
	if err != nil {
		return types.Collection{}, err
	}

	var collection types.Collection
	if meta, err := types.Unmarshal[types.Collection](object); err != nil {
		return types.Collection{}, err
	} else {
		collection = meta
	}
	collection.Slug = slug

	return collection, nil
}

func (s *S3Storage) deleteCollectionMetadata(ctx context.Context, slug string) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultS3Timeout)
	defer cancel()

	path := filepath.Join(s.MetadataPath, slug+".json")
	_, err := s.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(path),
	})

	return err
}
