package main

import (
	"bufio"
	"log"

	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/jacobmiller22/volume-backup/internal/config"
	"github.com/jacobmiller22/volume-backup/internal/crypto"
)

func configureAwsSession(envCfg *config.EnvironmentConfig, loc *config.Location) (*session.Session, error) {
	s, err := session.NewSession(
		aws.
			NewConfig().
			WithEndpoint(
				loc.S3_Endpoint,
			).
			WithRegion(
				loc.S3_Region,
			).
			WithCredentials(
				credentials.NewStaticCredentials(
					loc.S3_AccessKeyId,
					loc.S3_SecretAccessKey,
					"",
				),
			),
	)

	if err != nil {
		return nil, err
	}

	if envCfg.S3ForcePathStyle {
		s.Config.S3ForcePathStyle = aws.Bool(true)
	}

	return s, nil
}

func configurePuller(envCfg *config.EnvironmentConfig, source *config.Location) (Puller, error) {
	switch source.Kind {
	case "s3":

		sess, err := configureAwsSession(envCfg, source)
		if err != nil {
			return nil, err
		}

		return &S3PushPuller{
			s:      sess,
			bucket: source.S3_Bucket,
		}, nil
	case "fs":
		return &FsPushPuller{}, nil
	default:
		return nil, fmt.Errorf("invalid source kind")
	}
}

func configurePusher(envCfg *config.EnvironmentConfig, source *config.Location) (Pusher, error) {
	switch source.Kind {
	case "s3":

		sess, err := configureAwsSession(envCfg, source)
		if err != nil {
			return nil, err
		}

		return &S3PushPuller{
			s:      sess,
			bucket: source.S3_Bucket,
		}, nil
	case "fs":
		return &FsPushPuller{}, nil
	default:
		return nil, fmt.Errorf("invalid source kind")
	}
}

func main() {

	envCfg, err := config.LoadEnvironmentConfig()
	if err != nil {
		log.Fatalf("Error loading environment config: %v\n", err)
	}

	cfg, err := config.NewConfigLoader().WithFlagSet(flag.CommandLine, os.Args[1:]).Load()
	if err != nil {
		log.Fatalf("Error loading config: %v\n", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v\n", err)
	}

	puller, err := configurePuller(envCfg, &cfg.Source)
	if err != nil {
		log.Fatalf("Error setting up puller: %v\n", err)
	}
	pusher, err := configurePusher(envCfg, &cfg.Destination)
	if err != nil {
		log.Fatalf("Error setting up pusher: %v\n", err)
	}

	enc, err := crypto.NewStreamEncryptor(cfg.Encryption.Key)
	if err != nil {
		panic("unhandled error setting up stream encryptor")
	}
	dec, err := crypto.NewStreamDecryptor(cfg.Encryption.Key)
	if err != nil {
		panic("unhandled error setting up stream decryptor")
	}

	o := BackupOrchestrator{
		puller:          puller,
		pusher:          pusher,
		destinationPath: cfg.Destination.Path,
		encryptor:       enc,
		decryptor:       dec,
	}

	if cfg.Restore {
		if err := o.Restore(cfg.Source.Path); err != nil {
			log.Printf("Something went wrong while restoring path: %s; %v\n", cfg.Source.Path, err)
			os.Exit(1)
		}

	} else {
		if err := o.Backup(cfg.Source.Path); err != nil {
			log.Printf("Something went wrong while backing up path: %s; %v\n", cfg.Source.Path, err)
			os.Exit(1)
		}
	}
}

var ErrIsDir error = fmt.Errorf("cannot pull reader: is_directory")

type info struct {
	size int64
}

type Puller interface {
	// download
	Pull(path string) (io.Reader, error)
}

type Pusher interface {
	// upload
	Push(r io.Reader, path string) error
}

type FsPushPuller struct{}

func (p *FsPushPuller) Pull(path string) (io.Reader, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return fd, nil
}

func (p *FsPushPuller) Push(r io.Reader, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Reader from r and push to the path specified
	fd, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()

	w := bufio.NewWriter(fd)

	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}
	return w.Flush()
}

type S3PushPuller struct {
	s      *session.Session
	bucket string
}

func (p *S3PushPuller) Pull(path string) (io.Reader, error) {
	client := s3.New(p.s)
	input := &s3.GetObjectInput{
		Bucket: &p.bucket,
		Key:    &path,
	}
	output, err := client.GetObject(input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3, %w", err)
	}

	return output.Body, nil

}

func (p *S3PushPuller) Push(r io.Reader, path string) error {
	uploader := s3manager.NewUploader(p.s)

	upParams := s3manager.UploadInput{
		Bucket: &p.bucket,
		Key:    &path,
		Body:   r,
	}

	_, err := uploader.Upload(&upParams)

	return err
}

type BackupOrchestrator struct {
	puller          Puller
	pusher          Pusher
	destinationPath string
	encryptor       *crypto.StreamEncryptor
	decryptor       *crypto.StreamDecryptor
}

func (bo *BackupOrchestrator) Backup(path string) error {
	log.Printf("Backing up %s to %s\n", path, bo.destinationPath)

	// Pull
	r, err := bo.puller.Pull(path)
	if err != nil {
		return err
	}

	// Setup Encryption
	bo.encryptor.SetReader(r)

	// Push
	return bo.pusher.Push(bo.encryptor, bo.destinationPath)
}

func (bo *BackupOrchestrator) Restore(path string) error {
	log.Printf("Restoring from %s to %s\n", path, bo.destinationPath)

	r, err := bo.puller.Pull(path)
	if err != nil {
		return err
	}

	bo.decryptor.SetReader(r)

	return bo.pusher.Push(bo.decryptor, bo.destinationPath)
}
