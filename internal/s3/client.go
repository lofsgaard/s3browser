package s3client

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/lofsgaard/s3browser/internal/config"
)

type Client struct {
	s3     *s3.Client
	bucket string
}

func NewClient(cfg config.AppConfig) (*Client, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}
	if cfg.Profile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(cfg.Profile))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	s3Opts := []func(*s3.Options){}
	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		})
	}

	return &Client{
		s3:     s3.NewFromConfig(awsCfg, s3Opts...),
		bucket: cfg.Bucket,
	}, nil
}

func (c *Client) ListDir(ctx context.Context, prefix string, contToken string) ([]Entry, string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(c.bucket),
		Delimiter: aws.String("/"),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}
	if contToken != "" {
		input.ContinuationToken = aws.String(contToken)
	}

	out, err := c.s3.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, "", fmt.Errorf("list objects: %w", err)
	}

	var entries []Entry

	for _, cp := range out.CommonPrefixes {
		key := aws.ToString(cp.Prefix)
		name := strings.TrimPrefix(key, prefix)
		name = strings.TrimSuffix(name, "/")
		entries = append(entries, Entry{
			Name:    name,
			FullKey: key,
			Kind:    KindPrefix,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	var objects []Entry
	for _, obj := range out.Contents {
		key := aws.ToString(obj.Key)
		if key == prefix {
			continue
		}
		name := strings.TrimPrefix(key, prefix)
		sc := string(obj.StorageClass)
		if sc == "STANDARD" {
			sc = ""
		}
		objects = append(objects, Entry{
			Name:         name,
			FullKey:      key,
			Size:         aws.ToInt64(obj.Size),
			LastModified: aws.ToTime(obj.LastModified),
			StorageClass: sc,
			Kind:         KindObject,
		})
	}

	sort.Slice(objects, func(i, j int) bool {
		return objects[i].Name < objects[j].Name
	})

	entries = append(entries, objects...)

	var nextToken string
	if out.NextContinuationToken != nil {
		nextToken = *out.NextContinuationToken
	}

	return entries, nextToken, nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

func (c *Client) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	return out.Body, nil
}

func (c *Client) Upload(ctx context.Context, key string, r io.Reader, size int64) error {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(key),
		Body:          r,
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return fmt.Errorf("upload object: %w", err)
	}
	return nil
}
