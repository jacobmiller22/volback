package volback

import (
	"context"
	"io"

	"github.com/jacobmiller22/volume-backup/internal/config"

	"github.com/Backblaze/blazer/b2"
)

func newB2Cfg(ctx context.Context, loc *config.B2location) (*b2.Client, error) {
	client, err := b2.NewClient(ctx, loc.B2_ApplicationKeyId, loc.B2_ApplicationKey)
	if err != nil {
		return nil, err
	}

	return client, nil
}

type B2Pusher struct {
	b2client   *b2.Client
	bucketName string
}

func (p *B2Pusher) Push(r io.Reader, path string) error {
	ctx := context.TODO()

	bucket, err := p.b2client.Bucket(ctx, p.bucketName)
	if err != nil {
		return err
	}

	obj := bucket.Object(path)
	w := obj.NewWriter(ctx)
	defer w.Close()

	if _, err := io.Copy(w, r); err != nil {
		return err
	}

	return err
}
