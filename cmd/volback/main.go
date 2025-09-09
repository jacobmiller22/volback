package main

import (
	"log"

	"flag"
	"os"

	"github.com/jacobmiller22/volume-backup/internal/config"
	"github.com/jacobmiller22/volume-backup/internal/volback"
)

func main() {

	cfg, err := config.NewConfigLoader().WithFlagSet(flag.CommandLine, os.Args[1:]).Load()
	if err != nil {
		log.Fatalf("Error loading config: %v\n", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v\n", err)
	}

	executor, err := volback.NewExecutorFromConfig(cfg)
	if err != nil {
		log.Fatalf("Error setting up Executor: %s\n", err)
	}

	if cfg.Restore {
		if err := executor.Restore(); err != nil {
			log.Printf("Something went wrong while restoring path: %s; %v\n", cfg.Source.Path, err)
			os.Exit(1)
		}
	} else {
		if err := executor.Backup(); err != nil {
			log.Printf("Something went wrong while backing up path: %s; %v\n", cfg.Source.Path, err)
			os.Exit(1)
		}
	}

}
