package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	stdpath "path"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/jacobmiller22/volume-backup/zip"
)

type multiString []string

func (m *multiString) String() string {
	return fmt.Sprint(*m)
}

func (m *multiString) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func setupEncryptor(key string) (*cipher.Stream, error) {

	// Create a block cipher from a secret key
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}

	// Create a random nonce
	iv := make([]byte, block.BlockSize())
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	s := cipher.NewCTR(block, iv)
	return &s, nil
}

type config struct {
	Source struct {
		Paths multiString `json:"paths"`
	} `json:"source"`
	Encryption struct {
		Key string `json:"encryption_key"`
	}
	Destination struct {
		Prefix             string `json:"prefix"`
		Kind               string `json:"type"`
		S3_AccessKeyId     string `json:"s3_access_key_id"`
		S3_SecretAccessKey string `json:"s3_secret_access_key"`
		S3_Endpoint        string `json:"s3_endpoint"`
		S3_Bucket          string `json:"s3_bucket"`
		S3_Region          string `json:"s3_region"`
	} `json:"destination"`
}

func main() {

	var cfg config

	flag.Var(&cfg.Source.Paths, "path", "Path to a folder to backup.")
	flag.StringVar(&cfg.Encryption.Key, "enc.key", "", "The key to use for encryption")
	flag.StringVar(&cfg.Destination.Kind, "dest.kind", "s3", "the type of destination")
	flag.StringVar(&cfg.Destination.Prefix, "dest.prefix", "", "Path to a folder or file to backup.")
	flag.StringVar(&cfg.Destination.S3_Endpoint, "dest.endpoint", "", "Hostname to use as an endpoint for s3 compatible storage")
	flag.StringVar(&cfg.Destination.S3_Bucket, "dest.bucket", "", "Name of the bucket to backup to")
	flag.StringVar(&cfg.Destination.S3_AccessKeyId, "dest.access-key-id", "", "The access key id")
	flag.StringVar(&cfg.Destination.S3_SecretAccessKey, "dest.secret-access-key", "", "The secret access key")
	flag.StringVar(&cfg.Destination.S3_Region, "dest.region", "us-east-1", "The secret access key")

	flag.Parse()

	if cfg.Encryption.Key == "" {
		panic("Encryption key must be set!")
	}

	puller := &FsPushPuller{}

	sess, err := session.NewSession(
		aws.NewConfig().WithEndpoint(
			cfg.Destination.S3_Endpoint,
		).WithRegion(
			cfg.Destination.S3_Region,
		).WithCredentials(
			credentials.NewStaticCredentials(
				cfg.Destination.S3_AccessKeyId,
				cfg.Destination.S3_SecretAccessKey,
				"",
			),
		),
	)

	if err != nil {
		panic("error creating aws session")
	}

	pusher := &S3PushPuller{
		s:      sess,
		bucket: cfg.Destination.S3_Bucket,
	}

	s, err := setupEncryptor(cfg.Encryption.Key)

	if err != nil {
		panic("unhandled error setting up encryptor")
	}

	o := BackupOrchestrator{
		puller:    puller,
		pusher:    pusher,
		prefix:    cfg.Destination.Prefix,
		encryptor: s,
	}

	// For each path, follow destination config
	for _, p := range cfg.Source.Paths {
		if err := o.Backup(p); err != nil {
			fmt.Printf("Something went wrong while backing up path: %s; %v\n", p, err)
		}
	}
	fmt.Printf("Done backing up!!\n")
}

var ErrIsDir error = fmt.Errorf("cannot pull reader: is_directory")

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
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, ErrIsDir
	}

	return os.Open(path)
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
	return err
}

type S3PushPuller struct {
	s      *session.Session
	bucket string
}

func (p *S3PushPuller) Pull(path string) (io.Reader, error) {
	panic("not implemented")
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
	puller    Puller
	pusher    Pusher
	prefix    string
	encryptor *cipher.Stream
}

func (bo *BackupOrchestrator) Backup(path string) error {
	// Pull
	filename := stdpath.Base(path)

	r, err := bo.puller.Pull(path)
	isDir := errors.Is(err, ErrIsDir)
	if err != nil && !isDir {
		return err
	}

	// Zip
	if isDir {
		r, err = zip.ZipDir(path)
	} else {
		r, err = zip.ZipReader(r, filename)
	}
	if err != nil {
		return err
	}
	filename += ".zip"

	// Encrypt
	er := cipher.StreamReader{
		S: *bo.encryptor,
		R: r,
	}
	filename += ".enc"

	// Push
	destP := stdpath.Join(bo.prefix, filename)
	return bo.pusher.Push(er, destP)
}
