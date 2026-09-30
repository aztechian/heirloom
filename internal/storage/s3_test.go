package storage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aztechian/heirloom/internal/api/types"
	"github.com/aztechian/heirloom/internal/config"
	"github.com/aztechian/heirloom/internal/storage"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeS3Client is a minimal stand-in for *s3.Client that lets tests control
// the behavior of each S3 operation without making real network calls.
type fakeS3Client struct {
	headObjectFn    func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	putObjectFn     func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	getObjectFn     func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	deleteObjectFn  func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	listObjectsV2Fn func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)

	putObjectCalls []*s3.PutObjectInput
}

func (f *fakeS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if f.headObjectFn != nil {
		return f.headObjectFn(ctx, params, optFns...)
	}
	return nil, errors.New("HeadObject not stubbed")
}

func (f *fakeS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	f.putObjectCalls = append(f.putObjectCalls, params)
	if f.putObjectFn != nil {
		return f.putObjectFn(ctx, params, optFns...)
	}
	return &s3.PutObjectOutput{}, nil
}

func (f *fakeS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if f.getObjectFn != nil {
		return f.getObjectFn(ctx, params, optFns...)
	}
	return nil, errors.New("GetObject not stubbed")
}

func (f *fakeS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if f.deleteObjectFn != nil {
		return f.deleteObjectFn(ctx, params, optFns...)
	}
	return &s3.DeleteObjectOutput{}, nil
}

func (f *fakeS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if f.listObjectsV2Fn != nil {
		return f.listObjectsV2Fn(ctx, params, optFns...)
	}
	return &s3.ListObjectsV2Output{}, nil
}

func bodyOf(t *testing.T, v any) io.ReadCloser {
	t.Helper()
	data, err := types.Marshal(v)
	require.NoError(t, err)
	return io.NopCloser(bytes.NewReader(data))
}

func newS3Storage(client *fakeS3Client) *storage.S3Storage {
	logger := zerolog.Nop()
	return &storage.S3Storage{
		Logger:       &logger,
		Client:       client,
		Bucket:       "test-bucket",
		MetadataPath: "/" + storage.DefaultMetadataDir,
	}
}

func TestS3Storage_CollectionExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		client := &fakeS3Client{
			headObjectFn: func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return &s3.HeadObjectOutput{}, nil
			},
		}
		s := newS3Storage(client)
		assert.True(t, s.CollectionExists(context.Background(), "photos"))
	})

	t.Run("does not exist", func(t *testing.T) {
		client := &fakeS3Client{
			headObjectFn: func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return nil, errors.New("not found")
			},
		}
		s := newS3Storage(client)
		assert.False(t, s.CollectionExists(context.Background(), "photos"))
	})
}

func TestS3Storage_CreateCollection_New(t *testing.T) {
	client := &fakeS3Client{
		getObjectFn: func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return nil, errors.New("not found")
		},
	}
	s := newS3Storage(client)

	col := types.Collection{Name: "Photos", Slug: "photos"}
	require.NoError(t, s.CreateCollection(context.Background(), col))

	// One PutObject for the collection marker, one for the metadata sidecar.
	require.Len(t, client.putObjectCalls, 2)
	assert.Equal(t, "/photos/", *client.putObjectCalls[0].Key)
	assert.Equal(t, "/._meta/photos.json", *client.putObjectCalls[1].Key)
}

func TestS3Storage_CreateCollection_AlreadyExists(t *testing.T) {
	client := &fakeS3Client{
		getObjectFn: func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return &s3.GetObjectOutput{}, nil
		},
	}
	s := newS3Storage(client)

	require.NoError(t, s.CreateCollection(context.Background(), types.Collection{Slug: "photos"}))
	assert.Empty(t, client.putObjectCalls, "should not attempt to create an already-existing collection")
}

func TestS3Storage_CreateCollection_PutObjectError(t *testing.T) {
	wantErr := errors.New("put failed")
	client := &fakeS3Client{
		getObjectFn: func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return nil, errors.New("not found")
		},
		putObjectFn: func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			return nil, wantErr
		},
	}
	s := newS3Storage(client)

	err := s.CreateCollection(context.Background(), types.Collection{Slug: "photos"})
	assert.ErrorIs(t, err, wantErr)
}

func TestS3Storage_DeleteCollection_Success(t *testing.T) {
	var deletedKeys []string
	client := &fakeS3Client{
		deleteObjectFn: func(_ context.Context, params *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
			deletedKeys = append(deletedKeys, *params.Key)
			return &s3.DeleteObjectOutput{}, nil
		},
	}
	s := newS3Storage(client)

	require.NoError(t, s.DeleteCollection(context.Background(), types.Collection{Slug: "photos"}))
	assert.Equal(t, []string{"/photos/", "/._meta/photos.json"}, deletedKeys)
}

func TestS3Storage_DeleteCollection_Error(t *testing.T) {
	wantErr := errors.New("delete failed")
	client := &fakeS3Client{
		deleteObjectFn: func(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
			return nil, wantErr
		},
	}
	s := newS3Storage(client)

	err := s.DeleteCollection(context.Background(), types.Collection{Slug: "photos"})
	assert.ErrorIs(t, err, wantErr)
}

func TestS3Storage_ListCollections(t *testing.T) {
	client := &fakeS3Client{
		listObjectsV2Fn: func(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
			return &s3.ListObjectsV2Output{
				CommonPrefixes: []s3types.CommonPrefix{
					{Prefix: aws.String("/._meta/")},
					{Prefix: aws.String("/alpha/")},
					{Prefix: aws.String("/beta/")},
				},
			}, nil
		},
		getObjectFn: func(_ context.Context, params *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			switch *params.Key {
			case "/._meta/alpha.json":
				return &s3.GetObjectOutput{Body: bodyOf(t, types.Collection{Slug: "alpha", Name: "Alpha"})}, nil
			case "/._meta/beta.json":
				return &s3.GetObjectOutput{Body: bodyOf(t, types.Collection{Slug: "beta", Name: "Beta"})}, nil
			default:
				return nil, errors.New("not found")
			}
		},
	}
	s := newS3Storage(client)

	collections := s.ListCollections(context.Background())
	require.Len(t, collections, 2)

	slugs := []string{collections[0].Slug, collections[1].Slug}
	assert.ElementsMatch(t, []string{"alpha", "beta"}, slugs)
}

func TestS3Storage_ListCollections_ListError(t *testing.T) {
	client := &fakeS3Client{
		listObjectsV2Fn: func(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
			return nil, errors.New("list failed")
		},
	}
	s := newS3Storage(client)

	assert.Nil(t, s.ListCollections(context.Background()))
}

func TestS3Storage_ListCollections_SkipsUnreadableMetadata(t *testing.T) {
	client := &fakeS3Client{
		listObjectsV2Fn: func(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
			return &s3.ListObjectsV2Output{
				CommonPrefixes: []s3types.CommonPrefix{
					{Prefix: aws.String("/broken/")},
				},
			}, nil
		},
		getObjectFn: func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return nil, errors.New("not found")
		},
	}
	s := newS3Storage(client)

	assert.Empty(t, s.ListCollections(context.Background()))
}

func TestNewS3Storage(t *testing.T) {
	cfg := config.Config{
		S3: config.S3Config{
			Bucket:          "my-bucket",
			Region:          "us-east-1",
			AccessKeyID:     "AKIA_TEST",
			SecretAccessKey: "secret",
			Endpoint:        "http://localhost:9000",
		},
	}

	s := storage.NewS3Storage(context.Background(), cfg)
	require.NotNil(t, s)
	assert.Equal(t, "my-bucket", s.Bucket)
	assert.NotNil(t, s.Client)
	assert.Equal(t, "/"+storage.DefaultMetadataDir, s.MetadataPath)
}
