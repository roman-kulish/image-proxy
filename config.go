package roxy

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

const (
	clientTimeout      = 30 * time.Second
	serverReadTimeout  = 5 * time.Second
	serverWriteTimeout = 30 * time.Second

	envAddr               = "ROXY_ADDR"
	envSSLNoVerify        = "ROXY_SSL_NO_VERIFY"
	envClientTimeout      = "ROXY_CLIENT_TIMEOUT"
	envServerReadTimeout  = "ROXY_SERVER_READ_TIMEOUT"
	envServerWriteTimeout = "ROXY_SERVER_WRITE_TIMEOUT"
	envCacheDir           = "ROXY_CACHE_DIR"
)

type envVarParseError struct {
	Var string
	Err error
}

func (e envVarParseError) Error() string {
	return fmt.Sprintf("config: error parsing environment variable %s: %s", e.Var, e.Err)
}

type Config struct {
	// TCP address to listen on, ":http" if empty.
	Addr string

	// Whether an HTTP client should verify the Server's certificate chain and
	// host name.
	SSLNoVerify bool

	// Timeout specifies a time limit for requests made by an HTTP Client and
	// includes connection time, any redirects, and reading the response body.
	ClientTimeout time.Duration

	// ReadTimeout is the maximum duration for reading the entire request,
	// including the time from when the connection is accepted to when the
	// request body is fully read or to the end of the headers.
	ServerReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out writes of the
	// response.
	ServerWriteTimeout time.Duration

	// Path to a directory where cached files are stored.
	CacheDir string

	// Writer to use for logging access.
	AccessWriter io.Writer

	// Writer to use for logging server errors.
	ErrorWriter io.Writer
}

// NewConfig returns a new Config initialised with default configuration.
func NewConfig() *Config {
	return &Config{
		ClientTimeout:      clientTimeout,
		ServerReadTimeout:  serverReadTimeout,
		ServerWriteTimeout: serverWriteTimeout,
	}
}

// NewConfigFromEnv returns a new Config initialised with configuration from
// environment variables.
func NewConfigFromEnv() (*Config, error) {
	setDuration := func(key string, attr *time.Duration) error {
		if v := os.Getenv(key); v != "" {
			if i, err := strconv.Atoi(v); err != nil {
				return err
			} else if i > 0 { // if value is invalid, keep default
				*attr = time.Duration(i) * time.Second
			}
		}

		return nil
	}

	config := NewConfig()

	if addr := os.Getenv(envAddr); addr != "" {
		config.Addr = addr
	}

	if cacheDir := os.Getenv(envCacheDir); cacheDir != "" {
		config.CacheDir = cacheDir
	}

	if ssl := os.Getenv(envSSLNoVerify); ssl != "" {
		f, err := strconv.ParseBool(ssl)

		if err != nil {
			return nil, envVarParseError{envSSLNoVerify, err}
		}

		config.SSLNoVerify = f
	}

	attrs := map[string]*time.Duration{
		envClientTimeout:      &config.ClientTimeout,
		envServerReadTimeout:  &config.ServerReadTimeout,
		envServerWriteTimeout: &config.ServerWriteTimeout,
	}

	for k := range attrs {
		if err := setDuration(k, attrs[k]); err != nil {
			return nil, envVarParseError{k, err}
		}
	}

	return config, nil
}
