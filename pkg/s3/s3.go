package s3

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Client is a thin wrapper around the AWS SDK v2 S3 client tuned for SeaweedFS endpoints.
type Client struct {
	api     *s3.Client
	presign *s3.PresignClient
}

// NewClientFromEnv initialises a Client using environment variables expected by the project.
//
// Required environment variables:
//   - S3_ENDPOINT: host:port or full URL to the SeaweedFS S3 endpoint.
//   - S3_ACCESS_KEY / S3_SECRET_KEY: static credentials.
//
// Optional environment variables:
//   - S3_REGION (default "us-east-1").
//   - S3_DISABLE_TLS (bool; default false) to toggle TLS usage.
//   - S3_FORCE_PATH_STYLE (bool; default true).
func NewClientFromEnv() (*Client, error) {
	endpoint := strings.TrimSpace(os.Getenv("S3_ENDPOINT"))
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	region := os.Getenv("S3_REGION")
	if region == "" {
		region = "us-east-1"
	}

	if endpoint == "" {
		return nil, errors.New("S3_ENDPOINT is required")
	}
	if accessKey == "" || secretKey == "" {
		return nil, errors.New("S3_ACCESS_KEY and S3_SECRET_KEY are required")
	}

	disableTLS, _ := strconv.ParseBool(os.Getenv("S3_DISABLE_TLS"))
	forcePathStyle := true
	if v := strings.TrimSpace(os.Getenv("S3_FORCE_PATH_STYLE")); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			forcePathStyle = parsed
		}
	}

	scheme := "https"
	if disableTLS {
		scheme = "http"
	}

	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = fmt.Sprintf("%s://%s", scheme, endpoint)
	}

	ctx := context.Background()
	cfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		awsconfig.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = forcePathStyle
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &Client{
		api:     client,
		presign: s3.NewPresignClient(client),
	}, nil
}

// PutObject uploads data to the given bucket/key with checksum metadata.
func (c *Client) PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, sha256 string) error {
	if c == nil {
		return errors.New("nil client")
	}
	checksum, err := encodeSHA256(sha256)
	if err != nil {
		return err
	}

	_, err = c.api.PutObject(ctx, &s3.PutObjectInput{
		Bucket:            &bucket,
		Key:               &key,
		Body:              r,
		ContentLength:     &size,
		ChecksumAlgorithm: s3types.ChecksumAlgorithmSha256,
		ChecksumSHA256:    &checksum,
		Metadata: map[string]string{
			"sha256": sha256,
		},
	})
	return err
}

// PresignGet generates a presigned GET URL for the provided key and TTL.
func (c *Client) PresignGet(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	if c == nil {
		return "", errors.New("nil client")
	}

	req, err := c.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}, func(opts *s3.PresignOptions) {
		opts.Expires = ttl
	})
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

// PresignPut generates a presigned PUT URL for uploading an object within the provided TTL.
func (c *Client) PresignPut(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	if c == nil {
		return "", errors.New("nil client")
	}

	req, err := c.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}, func(opts *s3.PresignOptions) {
		opts.Expires = ttl
	})
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

func encodeSHA256(hexDigest string) (string, error) {
	if hexDigest == "" {
		return "", errors.New("sha256 digest required")
	}
	raw, err := hex.DecodeString(hexDigest)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}
