package volback

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jacobmiller22/volume-backup/internal/config"
	"github.com/jacobmiller22/volume-backup/internal/volback/transformers"

	"github.com/jacobmiller22/volume-backup/internal/crypto"
	"github.com/jacobmiller22/volume-backup/internal/pipes"
)

func NewExecutorFromConfig(cfg *config.Config) (*volbackExecutor, error) {

	var errs []error
	puller, err := pullerFromConfig(cfg)
	errs = append(errs, err)
	pusher, err := pusherFromConfig(cfg)
	errs = append(errs, err)

	encryptor, err := crypto.NewStreamEncryptor(cfg.Encryption.Key)
	errs = append(errs, err)
	if err != nil {
		panic("unhandled error setting up stream encryptor")
	}

	decryptor, err := crypto.NewStreamDecryptor(cfg.Encryption.Key)
	errs = append(errs, err)
	if err != nil {
		panic("unhandled error setting up stream decryptor")
	}

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	backupPipeline, err := pipes.NewIOPipeline([]pipes.IOPipe{
		pipes.NewIOPipe("encrypt", (&transformers.EncryptionTransformer{Encryptor: encryptor}).Transform),
	})
	if err != nil {
		return nil, fmt.Errorf("error setting up backup pipeline: %w", err)
	}
	restorePipeline, err := pipes.NewIOPipeline([]pipes.IOPipe{
		pipes.NewIOPipe("decrypt", (&transformers.DecryptionTransformer{Decryptor: decryptor}).Transform),
	})
	if err != nil {
		return nil, fmt.Errorf("error setting up restore pipeline: %w", err)
	}

	return &volbackExecutor{
		srcPath: cfg.Source.Path,
		puller:  puller,

		dstPath: cfg.Destination.Path,
		pusher:  pusher,

		backupPipeline:  backupPipeline,
		restorePipeline: restorePipeline,
	}, nil
}

type volbackExecutor struct {
	srcPath string
	puller  Puller

	dstPath string
	pusher  Pusher

	backupPipeline  *pipes.IOPipeline
	restorePipeline *pipes.IOPipeline
}

func process(ctx context.Context, puller Puller, srcPath string, pl *pipes.IOPipeline, pusher Pusher, dstPath string) error {
	initialReader, err := puller.Pull(srcPath)
	if err != nil {
		return err
	}

	r := pl.Execute(ctx, initialReader)

	if err := pusher.Push(r, dstPath); err != nil {
		return fmt.Errorf("error while pushing: %w", err)
	}

	return nil
}

func (e *volbackExecutor) Backup() error {
	log.Printf("Backing up %s to %s\n", e.srcPath, e.dstPath)
	return process(context.TODO(), e.puller, e.srcPath, e.backupPipeline, e.pusher, e.dstPath)
}

func (e *volbackExecutor) Restore() error {
	log.Printf("Restoring from %s to %s\n", e.srcPath, e.dstPath)
	return process(context.TODO(), e.puller, e.srcPath, e.restorePipeline, e.pusher, e.dstPath)
}
