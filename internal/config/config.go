package config

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/sethvargo/go-envconfig"
)

type EnvironmentConfig struct {
	S3ForcePathStyle bool `env:"S3_FORCE_PATH_STYLE,default=false"`
}

func LoadEnvironmentConfig() (*EnvironmentConfig, error) {
	var cfg EnvironmentConfig
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		return nil, fmt.Errorf("failed to process environment config: %w", err)
	}
	return &cfg, nil
}

func ConfigFromJsonPath(path string) (*Config, error) {

	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = json.NewDecoder(fd).Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func ConfigFromFlagset(flagset *flag.FlagSet, args []string) (*Config, error) {

	var cfg Config
	flagset.StringVar(&cfg.JsonConfigPath, "f", "", "Path to volback configuration file")
	flagset.StringVar(&cfg.Source.Kind, "src.kind", "", "the type of source")
	flagset.StringVar(&cfg.Source.Path, "src.path", "", "Path to a folder or file to backup.")
	flagset.StringVar(&cfg.Source.S3_Endpoint, "src.s3-endpoint", "", "Hostname to use as an endpoint for s3 compatible storage")
	flagset.StringVar(&cfg.Source.S3_Bucket, "src.s3-bucket", "", "Name of the bucket to source from")
	flagset.StringVar(&cfg.Source.S3_AccessKeyId, "src.s3-access-key-id", "", "The access key id")
	flagset.StringVar(&cfg.Source.S3_SecretAccessKey, "src.s3-secret-access-key", "", "The secret access key")
	flagset.StringVar(&cfg.Source.S3_Region, "src.s3-region", "", "The secret access key")

	flagset.BoolVar(&cfg.Restore, "restore", false, "If we should restore a backup")

	flagset.StringVar(&cfg.Encryption.Key, "enc.key", "", "The key to use for encryption")

	flagset.StringVar(&cfg.Destination.Kind, "dst.kind", "", "the type of destination")
	flagset.StringVar(&cfg.Destination.Path, "dst.path", "", "Path to place backup")
	flagset.StringVar(&cfg.Destination.S3_Endpoint, "dst.s3-endpoint", "", "Hostname to use as an endpoint for s3 compatible storage")
	flagset.StringVar(&cfg.Destination.S3_Bucket, "dst.s3-bucket", "", "Name of the bucket to backup to")
	flagset.StringVar(&cfg.Destination.S3_AccessKeyId, "dst.s3-access-key-id", "", "The access key id")
	flagset.StringVar(&cfg.Destination.S3_SecretAccessKey, "dst.s3-secret-access-key", "", "The secret access key")
	flagset.StringVar(&cfg.Destination.S3_Region, "dst.s3-region", "", "The secret access key")

	if err := flagset.Parse(args); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type configLoader struct {
	flagSet *flag.FlagSet
	args    []string
}

func NewConfigLoader() *configLoader {
	return &configLoader{}
}

func (cl *configLoader) WithFlagSet(flagset *flag.FlagSet, args []string) *configLoader {
	cl.flagSet = flagset
	cl.args = args
	return cl
}

func (cl *configLoader) Load() (*Config, error) {

	errs := make([]error, 2)

	configs := make([]*Config, 0, 2)

	flagCfg, err := ConfigFromFlagset(cl.flagSet, cl.args)
	if err != nil {
		return nil, err
	}
	configs = append(configs, flagCfg)

	jsonCfg, err := ConfigFromJsonPath(flagCfg.JsonConfigPath)

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if !errors.Is(err, os.ErrNotExist) {
		configs = append(configs, jsonCfg)
	}

	if err != nil {
		return nil, err
	}

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	return mergeConfigs(configs...), nil
}

type S3location struct {
	S3_AccessKeyId     string `json:"s3_access_key_id"`
	S3_SecretAccessKey string `json:"s3_secret_access_key"`
	S3_Endpoint        string `json:"s3_endpoint"`
	S3_Bucket          string `json:"s3_bucket"`
	S3_Region          string `json:"s3_region"`
}

type Location struct {
	Kind string `json:"kind"`
	Path string `json:"path"`

	S3location
}

type Config struct {
	JsonConfigPath string
	Source         Location `json:"source"`
	Restore        bool     `json:"restore"`
	Encryption     struct {
		Key string `json:"key"`
	} `json:"encryption"`
	Destination Location `json:"destination"`
}

// Merges the provided configs
// Configs provided later have high priority
func mergeConfigs(configs ...*Config) *Config {

	if len(configs) == 0 {
		return nil
	}

	C := configs[len(configs)-1]

	for _, c := range configs[:len(configs)-1] {
		// I'm lazy

		C.JsonConfigPath = weakAssign(C.JsonConfigPath, c.JsonConfigPath)

		C.Source.Kind = weakAssign(C.Source.Kind, c.Source.Kind)
		C.Source.Path = weakAssign(C.Source.Path, c.Source.Path)
		C.Source.S3_AccessKeyId = weakAssign(C.Source.S3_AccessKeyId, c.Source.S3_AccessKeyId)
		C.Source.S3_SecretAccessKey = weakAssign(C.Source.S3_SecretAccessKey, c.Source.S3_SecretAccessKey)
		C.Source.S3_Endpoint = weakAssign(C.Source.S3_Endpoint, c.Source.S3_Endpoint)
		C.Source.S3_Bucket = weakAssign(C.Source.S3_Bucket, c.Source.S3_Bucket)
		C.Source.S3_Region = weakAssign(C.Source.S3_Region, c.Source.S3_Region)

		C.Restore = weakAssign(C.Restore, c.Restore)

		C.Encryption.Key = weakAssign(C.Encryption.Key, c.Encryption.Key)

		C.Destination.Kind = weakAssign(C.Destination.Kind, c.Destination.Kind)
		C.Destination.Path = weakAssign(C.Destination.Path, c.Destination.Path)
		C.Destination.S3_AccessKeyId = weakAssign(C.Destination.S3_AccessKeyId, c.Destination.S3_AccessKeyId)
		C.Destination.S3_SecretAccessKey = weakAssign(C.Destination.S3_SecretAccessKey, c.Destination.S3_SecretAccessKey)
		C.Destination.S3_Endpoint = weakAssign(C.Destination.S3_Endpoint, c.Destination.S3_Endpoint)
		C.Destination.S3_Bucket = weakAssign(C.Destination.S3_Bucket, c.Destination.S3_Bucket)
		C.Destination.S3_Region = weakAssign(C.Destination.S3_Region, c.Destination.S3_Region)
	}

	return C
}

// return b if b is not the zero value for type T, else a
func weakAssign[T string | int | bool](a, b T) T {
	zero := *new(T)
	if b == zero {
		return a
	}
	return b
}

func (c *Config) Validate() error {
	if c.Source.Kind == "" {
		return fmt.Errorf("source kind is required")
	}
	if c.Destination.Kind == "" {
		return fmt.Errorf("destination kind is required")
	}
	if c.Encryption.Key == "" {
		return fmt.Errorf("encryption key is required")
	}
	if len(c.Encryption.Key) != 16 && len(c.Encryption.Key) != 24 && len(c.Encryption.Key) != 32 {
		return fmt.Errorf("encryption key must be 16, 24, or 32 bytes long")
	}

	return nil
}
