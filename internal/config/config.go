package config

import (
	"flag"
	"fmt"
	"os"
)

type AppConfig struct {
	Bucket   string
	Region   string
	Profile  string
	Endpoint string
}

func Parse() AppConfig {
	var cfg AppConfig

	flag.StringVar(&cfg.Bucket, "bucket", os.Getenv("S3_BUCKET"), "S3 bucket name (or set S3_BUCKET env var)")
	flag.StringVar(&cfg.Region, "region", envOrDefault("AWS_DEFAULT_REGION", "us-east-1"), "AWS region")
	flag.StringVar(&cfg.Profile, "profile", "", "AWS credentials profile")
	flag.StringVar(&cfg.Endpoint, "endpoint", "", "Custom endpoint URL (e.g. http://localhost:9000 for MinIO)")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: s3browser --bucket <bucket> [options]")
		fmt.Fprintln(os.Stderr)
		flag.PrintDefaults()
	}

	flag.Parse()

	if cfg.Bucket == "" {
		fmt.Fprintln(os.Stderr, "error: --bucket is required")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(1)
	}

	return cfg
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
