// Package s3 实现 S3 兼容对象存储驱动（同时用于 OSS / COS 的 S3 兼容端点）。
package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awstypes "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/storage"
)

// Storage 是 S3 兼容存储驱动。
type Storage struct {
	driverName string
	client     *awss3.Client
	bucket     string
	publicURL  string
	domain     string
}

// New 创建 S3 兼容驱动。
func New(driverName string, cfg config.ObjectStorageConfig, domain string) (*Storage, error) {
	if cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("%s storage: bucket/access_key/secret_key are required", driverName)
	}
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)
	if err != nil {
		return nil, err
	}
	endpoint := cfg.Endpoint
	if endpoint != "" && !strings.HasPrefix(endpoint, "http") {
		scheme := "https"
		if !cfg.UseSSL {
			scheme = "http"
		}
		endpoint = scheme + "://" + endpoint
	}
	client := awss3.NewFromConfig(awsCfg, func(o *awss3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		o.UsePathStyle = cfg.PathStyle
	})
	return &Storage{
		driverName: driverName,
		client:     client,
		bucket:     cfg.Bucket,
		publicURL:  cfg.PublicURL,
		domain:     domain,
	}, nil
}

// Driver 返回驱动名称。
func (s *Storage) Driver() string { return s.driverName }

// Put 上传对象。
func (s *Storage) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	in := &awss3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          r,
		ContentLength: aws.Int64(size),
	}
	if contentType != "" {
		in.ContentType = aws.String(contentType)
	}
	_, err := s.client.PutObject(ctx, in)
	return err
}

// Get 下载对象。
func (s *Storage) Get(ctx context.Context, key string) (io.ReadCloser, *storage.FileInfo, error) {
	out, err := s.client.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, nil, mapNotFound(err)
	}
	info := &storage.FileInfo{}
	if out.ContentLength != nil {
		info.Size = *out.ContentLength
	}
	if out.ContentType != nil {
		info.ContentType = *out.ContentType
	}
	if out.LastModified != nil {
		info.ModTime = *out.LastModified
	}
	return out.Body, info, nil
}

// Stat 获取对象元信息。
func (s *Storage) Stat(ctx context.Context, key string) (*storage.FileInfo, error) {
	out, err := s.client.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, mapNotFound(err)
	}
	info := &storage.FileInfo{}
	if out.ContentLength != nil {
		info.Size = *out.ContentLength
	}
	if out.ContentType != nil {
		info.ContentType = *out.ContentType
	}
	if out.LastModified != nil {
		info.ModTime = *out.LastModified
	}
	return info, nil
}

// Delete 删除对象。
func (s *Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// PublicURL 返回对象公开地址。
func (s *Storage) PublicURL(key string) string {
	base := s.publicURL
	if base == "" {
		base = s.domain
	}
	if base == "" {
		return ""
	}
	return join(base, key)
}

// Test 通过 HeadBucket 测试连通性。
func (s *Storage) Test(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, err := s.client.HeadBucket(ctx, &awss3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	return err
}

func mapNotFound(err error) error {
	var noSuchKey *awstypes.NoSuchKey
	var notFound *awstypes.NotFound
	if errors.As(err, &noSuchKey) || errors.As(err, &notFound) {
		return storage.ErrNotFound
	}
	var coded interface{ ErrorCode() string }
	if errors.As(err, &coded) && (coded.ErrorCode() == "NotFound" || coded.ErrorCode() == "NoSuchKey") {
		return storage.ErrNotFound
	}
	return err
}

func join(base, key string) string {
	for len(base) > 0 && base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}
	if key == "" {
		return base
	}
	if key[0] == '/' {
		return base + key
	}
	return base + "/" + key
}
