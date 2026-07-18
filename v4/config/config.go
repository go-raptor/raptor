package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
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
	Address           string   `yaml:"address"`
	Port              int      `yaml:"port"`
	ShutdownTimeout   int      `yaml:"shutdown_timeout"`
	ReadTimeout       int      `yaml:"read_timeout"`
	ReadHeaderTimeout int      `yaml:"read_header_timeout"`
	WriteTimeout      int      `yaml:"write_timeout"`
	IdleTimeout       int      `yaml:"idle_timeout"`
	MaxHeaderBytes    int      `yaml:"max_header_bytes"`
	MaxBodyBytes      int64    `yaml:"max_body_bytes"`
	IPExtractor       string   `yaml:"ip_extractor"`
	TrustedProxies    []string `yaml:"trusted_proxies"`
}

type DatabaseConfig struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	Name        string `yaml:"name"`
	AutoMigrate bool   `yaml:"auto_migrate"`
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
	DefaultServerConfigMaxBodyBytes      = int64(8 << 20) // explicit 0 disables the limit
	DefaultServerConfigIPExtractor       = "direct"
)

var (
	defaultConfigFiles = []string{".raptor.yaml", ".raptor.yml", ".raptor.conf"}
	devConfigFiles     = []string{".raptor.dev.yaml", ".raptor.dev.yml", ".raptor.dev.conf"}
	prodConfigFiles    = []string{".raptor.prod.yaml", ".raptor.prod.yml", ".raptor.prod.conf"}
	testConfigFiles    = []string{".raptor.test.yaml", ".raptor.test.yml", ".raptor.test.conf"}

	projectRootMarkers = slices.Concat(defaultConfigFiles, devConfigFiles, prodConfigFiles, testConfigFiles)
)

func (d DatabaseConfig) IsConfigured() bool {
	return d.Name != ""
}

func NewConfig(log *slog.Logger) (*Config, error) {
	return loadConfig(log, slices.Concat(defaultConfigFiles, prodConfigFiles, devConfigFiles))
}

func NewTestConfig(log *slog.Logger) (*Config, error) {
	return loadConfig(log, slices.Concat(defaultConfigFiles, testConfigFiles))
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
			MaxBodyBytes:      DefaultServerConfigMaxBodyBytes,
			IPExtractor:       DefaultServerConfigIPExtractor,
		},
		DatabaseConfig: DatabaseConfig{},
		AppConfig:      make(map[string]string),
	}
}

func findProjectRoot() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		for _, name := range projectRootMarkers {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				return dir, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func loadConfig(log *slog.Logger, configFiles []string) (*Config, error) {
	if root, ok := findProjectRoot(); ok {
		if err := os.Chdir(root); err != nil {
			log.Warn("Failed to change to project root", "root", root, "error", err)
		}
	}

	c := NewConfigDefaults()
	c.log = log

	var loadedFiles []string
	for _, file := range configFiles {
		err := c.loadConfigFromFile(file)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			c.log.Error("Failed to load configuration file", "file", file, "error", err)
			return c, err
		} else {
			loadedFiles = append(loadedFiles, file)
			c.log.Info("Configuration loaded", "file", file)
		}
	}

	if len(loadedFiles) == 0 {
		log.Warn("No configuration files found, using defaults")
	}
	if containsAny(loadedFiles, devConfigFiles) && containsAny(loadedFiles, prodConfigFiles) {
		log.Warn("Both dev and prod configuration files are present; dev values override prod")
	}

	c.applyEnvironmentVariables()
	c.applyAppEnvironmentVariables("APP_")

	return c, nil
}

func containsAny(haystack, needles []string) bool {
	for _, needle := range needles {
		if slices.Contains(haystack, needle) {
			return true
		}
	}
	return false
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

func (c *Config) applyEnvironmentVariables() {
	c.applyEnvironmentVariable("GENERAL_LOG_LEVEL", &c.GeneralConfig.LogLevel)

	c.applyEnvironmentVariable("SERVER_ADDRESS", &c.ServerConfig.Address)
	c.applyEnvironmentVariable("SERVER_PORT", &c.ServerConfig.Port)
	c.applyEnvironmentVariable("SERVER_SHUTDOWN_TIMEOUT", &c.ServerConfig.ShutdownTimeout)
	c.applyEnvironmentVariable("SERVER_READ_TIMEOUT", &c.ServerConfig.ReadTimeout)
	c.applyEnvironmentVariable("SERVER_READ_HEADER_TIMEOUT", &c.ServerConfig.ReadHeaderTimeout)
	c.applyEnvironmentVariable("SERVER_WRITE_TIMEOUT", &c.ServerConfig.WriteTimeout)
	c.applyEnvironmentVariable("SERVER_IDLE_TIMEOUT", &c.ServerConfig.IdleTimeout)
	c.applyEnvironmentVariable("SERVER_MAX_HEADER_BYTES", &c.ServerConfig.MaxHeaderBytes)
	c.applyEnvironmentVariable("SERVER_MAX_BODY_BYTES", &c.ServerConfig.MaxBodyBytes)
	c.applyEnvironmentVariable("SERVER_IP_EXTRACTOR", &c.ServerConfig.IPExtractor)
	c.applyEnvironmentVariable("SERVER_TRUSTED_PROXIES", &c.ServerConfig.TrustedProxies)

	c.applyEnvironmentVariable("DATABASE_HOST", &c.DatabaseConfig.Host)
	c.applyEnvironmentVariable("DATABASE_PORT", &c.DatabaseConfig.Port)
	c.applyEnvironmentVariable("DATABASE_USERNAME", &c.DatabaseConfig.Username)
	c.applyEnvironmentVariable("DATABASE_PASSWORD", &c.DatabaseConfig.Password)
	c.applyEnvironmentVariable("DATABASE_NAME", &c.DatabaseConfig.Name)
	c.applyEnvironmentVariable("DATABASE_AUTO_MIGRATE", &c.DatabaseConfig.AutoMigrate)
}

func (c *Config) applyEnvironmentVariable(key string, value interface{}) {
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
			} else {
				c.log.Warn("Invalid environment variable value, ignoring", "key", key, "value", env)
			}
		case *int:
			if number, err := strconv.Atoi(env); err == nil {
				*v = number
			} else {
				c.log.Warn("Invalid environment variable value, ignoring", "key", key, "value", env)
			}
		case *int64:
			if number, err := strconv.ParseInt(env, 10, 64); err == nil {
				*v = number
			} else {
				c.log.Warn("Invalid environment variable value, ignoring", "key", key, "value", env)
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

	if hasURLUserinfo(valueStr) {
		return "********"
	}

	return valueStr
}

// hasURLUserinfo reports whether value looks like a URL carrying
// credentials (scheme://user:pass@host), e.g. a database DSN.
func hasURLUserinfo(value string) bool {
	if !strings.Contains(value, "://") || !strings.Contains(value, "@") {
		return false
	}
	u, err := url.Parse(value)
	return err == nil && u.User != nil
}
