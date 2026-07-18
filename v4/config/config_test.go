package config

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, nil))
}

func writeConfigFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMaskSensitiveDataByKey(t *testing.T) {
	if got := maskSensitiveData("DATABASE_PASSWORD", "hunter2"); got != "********" {
		t.Fatalf("password value should be masked: got %v", got)
	}
	if got := maskSensitiveData("SERVER_PORT", "3000"); got != "3000" {
		t.Fatalf("non-sensitive value should pass through: got %v", got)
	}
}

func TestMaskURLUserinfo(t *testing.T) {
	if got := maskSensitiveData("APP_DATABASE_URL", "postgres://user:hunter2@host:5432/db"); got != "********" {
		t.Fatalf("URL with embedded credentials should be masked: got %v", got)
	}
	if got := maskSensitiveData("APP_SITE_URL", "https://example.com/path"); got != "https://example.com/path" {
		t.Fatalf("URL without credentials should pass through: got %v", got)
	}
}

func TestInvalidEnvValueWarnsAndKeepsDefault(t *testing.T) {
	var buf bytes.Buffer
	c := NewConfigDefaults()
	c.log = testLogger(&buf)
	t.Setenv("SERVER_PORT", "not-a-number")

	c.applyEnvirontmentVariables()

	if c.ServerConfig.Port != DefaultServerConfigPort {
		t.Fatalf("invalid value must keep the default port: got %d", c.ServerConfig.Port)
	}
	if !strings.Contains(buf.String(), "Invalid") {
		t.Fatalf("invalid env value should log a warning, got log: %s", buf.String())
	}
}

func TestFilePrecedenceDevOverridesDefault(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, dir, ".raptor.yaml", "server:\n  port: 1111\n")
	writeConfigFile(t, dir, ".raptor.dev.yaml", "server:\n  port: 2222\n")
	t.Chdir(dir)

	var buf bytes.Buffer
	cfg, err := NewConfig(testLogger(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerConfig.Port != 2222 {
		t.Fatalf("dev config should override the default file: got %d", cfg.ServerConfig.Port)
	}
}

func TestDevAndProdBothPresentWarns(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, dir, ".raptor.prod.yaml", "server:\n  port: 1111\n")
	writeConfigFile(t, dir, ".raptor.dev.yaml", "server:\n  port: 2222\n")
	t.Chdir(dir)

	var buf bytes.Buffer
	if _, err := NewConfig(testLogger(&buf)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "dev") || !strings.Contains(strings.ToLower(buf.String()), "warn") {
		t.Fatalf("having both dev and prod configs should log a warning, got log: %s", buf.String())
	}
}

func TestEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, dir, ".raptor.yaml", "server:\n  port: 1111\n")
	t.Chdir(dir)
	t.Setenv("SERVER_PORT", "3333")

	var buf bytes.Buffer
	cfg, err := NewConfig(testLogger(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerConfig.Port != 3333 {
		t.Fatalf("environment must override files: got %d", cfg.ServerConfig.Port)
	}
}

func TestAppEnvironmentVariablesApplied(t *testing.T) {
	var buf bytes.Buffer
	c := NewConfigDefaults()
	c.log = testLogger(&buf)
	t.Setenv("APP_FEATURE_FLAG", "on")

	c.applyAppEnvironmentVariables("APP_")

	if got := c.AppConfig["feature_flag"]; got != "on" {
		t.Fatalf("APP_ vars should land in AppConfig lowercased: got %q", got)
	}
}

func TestMergeConfigSkipsZeroValues(t *testing.T) {
	dst := NewConfigDefaults()
	src := &Config{ServerConfig: ServerConfig{Port: 9999}}

	MergeConfig(dst, src)

	if dst.ServerConfig.Port != 9999 {
		t.Fatalf("non-zero src values must override: got %d", dst.ServerConfig.Port)
	}
	if dst.ServerConfig.MaxBodyBytes != DefaultServerConfigMaxBodyBytes {
		t.Fatalf("zero src values must not clobber defaults: got %d", dst.ServerConfig.MaxBodyBytes)
	}
}
