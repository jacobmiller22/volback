package volback

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jacobmiller22/volume-backup/internal/config"
)

type Puller interface {
	// download
	Pull(path string) (io.Reader, error)
}

func pullerFromConfig(cfg *config.Config) (Puller, error) {
	switch cfg.Source.Kind {
	case "s3":
		awsCfg, err := newAwsCfg(&cfg.Source)
		if err != nil {
			return nil, err
		}

		return &S3PushPuller{
			s3client: s3.NewFromConfig(*awsCfg, func(o *s3.Options) { o.UsePathStyle = cfg.S3ForcePathStyle }),
			bucket:   cfg.Source.S3_Bucket,
		}, nil
	case "fs":
		return &FsPushPuller{
			restore: cfg.Restore,
		}, nil
	default:
		return nil, fmt.Errorf("invalid source kind")
	}
}
