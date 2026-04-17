package config

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	log *slog.Logger

	GeneralConfig  GeneralConfig     `yaml:"general"`
	ServerConfig   ServerConfig      `yaml:"server"`
	DatabaseConfig DatabaseConfig    `yaml:"database"`
	AppConfig      map[string]string `yaml:"app"`
}

type GeneralConfig struct {
	LogLevel string `yaml:"log_level"`
}

type ServerConfig struct {
	Address           string `yaml:"address"`
	Port              int    `yaml:"port"`
	ShutdownTimeout   int    `yaml:"shutdown_timeout"`
	ReadTimeout       int    `yaml:"read_timeout"`
	ReadHeaderTimeout int    `yaml:"read_header_timeout"`
	WriteTimeout      int    `yaml:"write_timeout"`
	IdleTimeout       int    `yaml:"idle_timeout"`
	MaxHeaderBytes    int    `yaml:"max_header_bytes"`
	IPExtractor       string `yaml:"ip_extractor"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

func (d DatabaseConfig) IsConfigured() bool {
	return d.Host != "" && d.Username != "" && d.Name != ""
}

const (
	DefaultGeneralConfigLogLevel = "info"

	DefaultServerConfigAddress           = "127.0.0.1"
	DefaultServerConfigPort              = 3000
	DefaultServerConfigShutdownTimeout   = 3
	DefaultServerConfigReadTimeout       = 0
	DefaultServerConfigReadHeaderTimeout = 10
	DefaultServerConfigWriteTimeout      = 0
	DefaultServerConfigIdleTimeout       = 120
	DefaultServerConfigMaxHeaderBytes    = 1 << 20
	DefaultServerConfigIPExtractor       = "direct"
)

func NewConfig(log *slog.Logger) (*Config, error) {
	return loadConfig(log, []string{
		".raptor.yaml",
		".raptor.yml",
		".raptor.conf",
		".raptor.prod.yaml",
		".raptor.prod.yml",
		".raptor.prod.conf",
		".raptor.dev.yaml",
		".raptor.dev.yml",
		".raptor.dev.conf",
	})
}

func NewTestConfig(log *slog.Logger) (*Config, error) {
	return loadConfig(log, []string{
		".raptor.yaml",
		".raptor.yml",
		".raptor.conf",
		".raptor.test.yaml",
		".raptor.test.yml",
		".raptor.test.conf",
	})
}

func NewConfigDefaults() *Config {
	return &Config{
		GeneralConfig: GeneralConfig{
			LogLevel: DefaultGeneralConfigLogLevel,
		},
		ServerConfig: ServerConfig{
			Address:           DefaultServerConfigAddress,
			Port:              DefaultServerConfigPort,
			ShutdownTimeout:   DefaultServerConfigShutdownTimeout,
			ReadTimeout:       DefaultServerConfigReadTimeout,
			ReadHeaderTimeout: DefaultServerConfigReadHeaderTimeout,
			WriteTimeout:      DefaultServerConfigWriteTimeout,
			IdleTimeout:       DefaultServerConfigIdleTimeout,
			MaxHeaderBytes:    DefaultServerConfigMaxHeaderBytes,
			IPExtractor:       DefaultServerConfigIPExtractor,
		},
		DatabaseConfig: DatabaseConfig{},
		AppConfig:      make(map[string]string),
	}
}

func loadConfig(log *slog.Logger, configFiles []string) (*Config, error) {
	c := NewConfigDefaults()
	c.log = log

	loaded := false
	for _, file := range configFiles {
		err := c.loadConfigFromFile(file)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			c.log.Error("Failed to load configuration file", "file", file, "error", err)
			return c, err
		} else {
			loaded = true
			c.log.Info("Configuration loaded", "file", file)
		}
	}

	if !loaded {
		log.Warn("No configuration files found, using defaults")
	}

	c.applyEnvirontmentVariables()
	c.applyAppEnvironmentVariables("APP_")

	return c, nil
}

func MergeConfig(dst, src *Config) {
	if src == nil {
		return
	}

	srcVal := reflect.ValueOf(src).Elem()
	dstVal := reflect.ValueOf(dst).Elem()

	for i := 0; i < srcVal.NumField(); i++ {
		fieldName := srcVal.Type().Field(i).Name
		if fieldName == "log" {
			continue
		}

		srcField := srcVal.Field(i)
		dstField := dstVal.Field(i)

		if fieldName == "AppConfig" && srcField.Len() > 0 {
			for _, key := range srcField.MapKeys() {
				dstField.SetMapIndex(key, srcField.MapIndex(key))
			}
			continue
		}

		mergeConfigValues(dstField, srcField)
	}
}

func mergeConfigValues(dst, src reflect.Value) {
	if src.Kind() == reflect.Struct {
		for i := 0; i < src.NumField(); i++ {
			srcField := src.Field(i)
			dstField := dst.Field(i)

			if srcField.Kind() == reflect.Struct {
				mergeConfigValues(dstField, srcField)
			} else if !srcField.IsZero() {
				dstField.Set(srcField)
			}
		}
	} else if !src.IsZero() {
		dst.Set(src)
	}
}

func (c *Config) loadConfigFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("malformed YAML in config file %s: %w", path, err)
	}

	return nil
}

func (c *Config) applyEnvirontmentVariables() {
	c.applyEnvirontmentVariable("GENERAL_LOG_LEVEL", &c.GeneralConfig.LogLevel)

	c.applyEnvirontmentVariable("SERVER_ADDRESS", &c.ServerConfig.Address)
	c.applyEnvirontmentVariable("SERVER_PORT", &c.ServerConfig.Port)
	c.applyEnvirontmentVariable("SERVER_SHUTDOWN_TIMEOUT", &c.ServerConfig.ShutdownTimeout)
	c.applyEnvirontmentVariable("SERVER_READ_TIMEOUT", &c.ServerConfig.ReadTimeout)
	c.applyEnvirontmentVariable("SERVER_READ_HEADER_TIMEOUT", &c.ServerConfig.ReadHeaderTimeout)
	c.applyEnvirontmentVariable("SERVER_WRITE_TIMEOUT", &c.ServerConfig.WriteTimeout)
	c.applyEnvirontmentVariable("SERVER_IDLE_TIMEOUT", &c.ServerConfig.IdleTimeout)
	c.applyEnvirontmentVariable("SERVER_MAX_HEADER_BYTES", &c.ServerConfig.MaxHeaderBytes)
	c.applyEnvirontmentVariable("SERVER_IP_EXTRACTOR", &c.ServerConfig.IPExtractor)

	c.applyEnvirontmentVariable("DATABASE_HOST", &c.DatabaseConfig.Host)
	c.applyEnvirontmentVariable("DATABASE_PORT", &c.DatabaseConfig.Port)
	c.applyEnvirontmentVariable("DATABASE_USERNAME", &c.DatabaseConfig.Username)
	c.applyEnvirontmentVariable("DATABASE_PASSWORD", &c.DatabaseConfig.Password)
	c.applyEnvirontmentVariable("DATABASE_NAME", &c.DatabaseConfig.Name)
}

func (c *Config) applyEnvirontmentVariable(key string, value interface{}) {
	if env, ok := os.LookupEnv(key); ok {
		c.log.Info("Applying environment variable", "key", key, "value", maskSensitiveData(key, env))
		switch v := value.(type) {
		case *string:
			*v = env
		case *bool:
			if env == "true" || env == "1" {
				*v = true
			} else if env == "false" || env == "0" {
				*v = false
			}
		case *int:
			if number, err := strconv.Atoi(env); err == nil {
				*v = number
			}
		case *[]string:
			*v = strings.Split(env, ",")
		default:
		}
	}
}

func (c *Config) applyAppEnvironmentVariables(prefix string) {
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, prefix) {
			continue
		}
		key, value, found := strings.Cut(kv, "=")
		if !found {
			continue
		}
		c.log.Info("Applying app environment variable", "key", key, "value", maskSensitiveData(key, value))
		key = strings.ToLower(strings.TrimPrefix(key, prefix))
		c.AppConfig[key] = value
	}
}

func maskSensitiveData(key string, value interface{}) interface{} {
	valueStr, ok := value.(string)
	if !ok {
		return value
	}

	sensitiveWords := []string{"password", "token", "key", "secret", "auth"}
	keyLower := strings.ToLower(key)
	for _, word := range sensitiveWords {
		if strings.Contains(keyLower, word) {
			return "********"
		}
	}

	return valueStr
}
