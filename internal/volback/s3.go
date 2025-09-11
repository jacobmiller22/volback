package volback

import (
	"context"
	"fmt"
	"io"

	"github.com/jacobmiller22/volume-backup/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscredentials "github.com/aws/aws-sdk-go-v2/credentials"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func newAwsCfg(loc *config.Location) (*aws.Config, error) {

	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithCredentialsProvider(
			awscredentials.NewStaticCredentialsProvider(
				loc.S3_AccessKeyId,
				loc.S3_SecretAccessKey,
				"",
			),
		),
		awsconfig.WithBaseEndpoint(loc.S3_Endpoint),
		awsconfig.WithRegion(loc.S3_Region),
	)

	if err != nil {
		return nil, fmt.Errorf("error loading default aws config: %w", err)
	}

	return &awsCfg, nil
}

type S3PushPuller struct {
	s3client *s3.Client
	bucket   string
}

func (p *S3PushPuller) Pull(path string) (io.Reader, error) {

	input := &s3.GetObjectInput{
		Bucket: &p.bucket,
		Key:    &path,
	}
	output, err := p.s3client.GetObject(context.Background(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3, %w", err)
	}

	return output.Body, nil

}

func (p *S3PushPuller) Push(r io.Reader, path string) error {

	uploader := s3manager.NewUploader(p.s3client)

	upParams := s3.PutObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
		Body:   r,
	}

	_, err := uploader.Upload(context.Background(), &upParams)

	return err
}
