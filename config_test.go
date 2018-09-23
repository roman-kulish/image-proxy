package roxy

import (
	"os"
	"testing"
	"time"
)

var envVars = [...]string{
	envAddr,
	envSSLNoVerify,
	envClientTimeout,
	envServerReadTimeout,
	envServerWriteTimeout,
	envCacheDir,
}

func clearEnv() {
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}

func assertDuration(t *testing.T, field string, expected, got time.Duration) {
	t.Helper()

	if expected != got {
		t.Errorf("failed assertion for Config field %s: expected %v, got %v", field, expected, got)
	}
}

func TestNewConfig(t *testing.T) {

	c := NewConfig()

	assertDuration(t, "ClientTimeout", c.ClientTimeout, clientTimeout)
	assertDuration(t, "ServerReadTimeout", c.ServerReadTimeout, serverReadTimeout)
	assertDuration(t, "ServerWriteTimeout", c.ServerWriteTimeout, serverWriteTimeout)
}

func TestNewConfigFromEnv(t *testing.T) {
	expAddr := ":80"
	expCacheDir := "/tmp/roxy"

	env := map[string]string{
		envAddr:               expAddr,
		envSSLNoVerify:        "true",
		envClientTimeout:      "1",
		envServerReadTimeout:  "1",
		envServerWriteTimeout: "10",
		envCacheDir:           expCacheDir,
	}

	for k := range env {
		os.Setenv(k, env[k])
	}

	defer clearEnv()

	c, err := NewConfigFromEnv()

	if err != nil {
		t.Fatal(err)
	}

	if c.Addr != expAddr {
		t.Errorf("expected Config.Addr to be %s, got %s", c.Addr, expAddr)
	}

	if c.SSLNoVerify != true {
		t.Errorf("expected Config.SSLNoVerify to be true, got %v", c.SSLNoVerify)
	}

	assertDuration(t, "ClientTimeout", c.ClientTimeout, 1*time.Second)
	assertDuration(t, "ServerReadTimeout", c.ServerReadTimeout, 1*time.Second)
	assertDuration(t, "ServerWriteTimeout", c.ServerWriteTimeout, 10*time.Second)

	if c.CacheDir != expCacheDir {
		t.Errorf("expected Config.CacheDir to be %s, got %s", c.CacheDir, expCacheDir)
	}
}

func TestNewConfigFromEnvInvalidDurations(t *testing.T) {
	env := map[string]string{
		envClientTimeout:      "-1",
		envServerReadTimeout:  "0",
		envServerWriteTimeout: "",
	}

	for k := range env {
		os.Setenv(k, env[k])
	}

	defer clearEnv()

	c, err := NewConfigFromEnv()

	if err != nil {
		t.Fatal(err)
	}

	assertDuration(t, "ClientTimeout", c.ClientTimeout, clientTimeout)
	assertDuration(t, "ServerReadTimeout", c.ServerReadTimeout, serverReadTimeout)
	assertDuration(t, "ServerWriteTimeout", c.ServerWriteTimeout, serverWriteTimeout)
}
