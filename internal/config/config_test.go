package config

import (
	"flag"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestConfigFromEnv(t *testing.T) {
	// Set environment variable for testing
	os.Setenv("S3_FORCE_PATH_STYLE", "true")
	defer os.Unsetenv("S3_FORCE_PATH_STYLE")

	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("Failed to load environment config: %v", err)
	}

	expectedCfg := &Config{
		S3ForcePathStyle: true,
	}

	if diff := cmp.Diff(expectedCfg, cfg); diff != "" {
		t.Errorf("EnvironmentConfig mismatch (-expected +actual):\n%s", diff)
	}
}

func TestNewConfig(t *testing.T) {
	var jsonData string = `
		{
			"source": {
				"kind": "fs",
				"path": "Makefile"
			},
			"restore": false,
			"encryption": {
				"key": "temp size 16 key"
			},
			"destination": {
				"kind": "s3",
				"path": "testing/backups/Makefile.backup",
				"s3_access_key_id": "keyid",
				"s3_secret_access_key": "secretkey",
				"s3_endpoint": "s3.us-east-005.backblazeb2.com",
				"s3_bucket": "jacobmiller22-secure-backup",
				"s3_region": "us-east-1"
			}
		}`

	// Add to testing temp file
	tmpFile := t.TempDir() + "/config.json"
	err := os.WriteFile(tmpFile, []byte(jsonData), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp config file: %v", err)
	}

	flagset := flag.NewFlagSet("test", flag.ContinueOnError)

	//

	loader := NewConfigLoader().WithFlagSet(flagset, []string{"-f", tmpFile, "-src.s3-endpoint", "overridden-endpoint"})

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	expectedCfg := &Config{
		JsonConfigPath: tmpFile,
		Source: Location{
			Kind: "fs",
			Path: "Makefile",
			S3location: S3location{
				S3_Endpoint: "overridden-endpoint",
			},
		},
		Restore: false,
		Encryption: struct {
			Key string `json:"key"`
		}{
			Key: "temp size 16 key",
		},
		Destination: Location{
			Kind: "s3",
			Path: "testing/backups/Makefile.backup",
			S3location: S3location{
				S3_AccessKeyId:     "keyid",
				S3_SecretAccessKey: "secretkey",
				S3_Endpoint:        "s3.us-east-005.backblazeb2.com",
				S3_Bucket:          "jacobmiller22-secure-backup",
				S3_Region:          "us-east-1",
			},
		},
	}

	if diff := cmp.Diff(expectedCfg, cfg); diff != "" {
		t.Errorf("Config mismatch (-expected +actual):\n%s", diff)
	}
}
