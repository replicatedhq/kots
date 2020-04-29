package s3

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
)

func GetConfig() *aws.Config {
	forcePathStyle := false
	if os.Getenv("S3_BUCKET_ENDPOINT") == "true" {
		forcePathStyle = true
	}

	region := os.Getenv("S3_REGION")
	if region == "" {
		region = "us-east-1"
	}

	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(os.Getenv("S3_ACCESS_KEY_ID"), os.Getenv("S3_SECRET_ACCESS_KEY"), ""),
		Endpoint:         aws.String(os.Getenv("S3_ENDPOINT")),
		Region:           aws.String(region),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(forcePathStyle),
	}

	return s3Config
}

func BucketName() string {
	return os.Getenv("S3_BUCKET_NAME")
}
