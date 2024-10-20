package backup

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Uploader struct {
	S3Client *s3.Client
	Bucket   string
}

func NewS3Uploader(bucket string, isProd bool) (*S3Uploader, error) {
	var cfg aws.Config
	if isProd {
		c, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-west-2"))
		if err != nil {
			return nil, fmt.Errorf("unable to load SDK config, %v", err)
		}
		cfg = c
	} else {
		c, err := config.LoadDefaultConfig(context.TODO(), config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, region string) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           "http://localhost:4566", // LocalStack endpoint
					SigningRegion: "eu-west-2",             // Set your desired region
				}, nil
			},
		)))
		if err != nil {
			return nil, fmt.Errorf("unable to load SDK config, %v", err)
		}
		cfg = c
	}

	// this is needed to write backup files in localstack
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &S3Uploader{
		S3Client: s3Client,
		Bucket:   bucket,
	}, nil
}

func (uploader *S3Uploader) UploadFile(filePath string, s3FilePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %v: %v", filePath, err)
	}
	defer file.Close()

	_, err = uploader.S3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(uploader.Bucket),
		Key:    aws.String(s3FilePath),
		Body:   file,
		ACL:    types.ObjectCannedACLPrivate,
	})

	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %v", err)
	}

	return nil
}
