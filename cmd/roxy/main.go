package main

import (
	"fmt"
	. "github.com/roman-kulish/image-proxy"
	"log"
	"os"
	"path/filepath"
)

const cacheSubDir = "roxy"

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	config, err := NewConfigFromEnv()

	if err != nil {
		logger.Fatalf("Error while creating configuration: %s", err)
	}

	if config.CacheDir == "" {
		if dir, err := findCacheDirectory(); err != nil {
			logger.Fatalf("Error setting up cache directory: %s", err)
		} else {
			config.CacheDir = dir
		}
	}

	logger.Printf("Using cache directory: %s", config.CacheDir)
	logger.Printf("Listening on %s", config.Addr)

	if err := NewServer(config).Start(); err != nil {
		logger.Fatalf("Error while starting the server: %s", err)
	}
}

// findCacheDirectory attempts to find a suitable location to store cached
// files.
func findCacheDirectory() (string, error) {
	dir, err := os.UserCacheDir()

	if err != nil {
		dir = os.TempDir()
	}

	dir = filepath.Join(dir, cacheSubDir)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("error creating cache diretory %s: %s", dir, err)
	}

	return dir, nil
}
