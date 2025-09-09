package volback

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jacobmiller22/volume-backup/internal/config"
)

type Pusher interface {
	// upload
	Push(r io.Reader, path string) error
}

func pusherFromConfig(cfg *config.Config) (Pusher, error) {

	switch cfg.Destination.Kind {
	case "s3":

		awsCfg, err := newAwsCfg(&cfg.Destination)
		if err != nil {
			return nil, err
		}

		return &S3PushPuller{
			s3client: s3.NewFromConfig(*awsCfg, func(o *s3.Options) { o.UsePathStyle = cfg.S3ForcePathStyle }),
			bucket:   cfg.Destination.S3_Bucket,
		}, nil
	case "fs":
		return &FsPushPuller{restore: cfg.Restore}, nil
	default:
		return nil, fmt.Errorf("invalid source kind")
	}
}
