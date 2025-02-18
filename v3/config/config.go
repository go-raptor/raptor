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
	log            *slog.Logger
	GeneralConfig  GeneralConfig     `yaml:"general"`
	ServerConfig   ServerConfig      `yaml:"server"`
	DatabaseConfig DatabaseConfig    `yaml:"database"`
	StaticConfig   StaticConfig      `yaml:"static"`
	CORSConfig     CORSConfig        `yaml:"cors"`
	AppConfig      map[string]string `yaml:"app"`
}

type GeneralConfig struct {
	LogLevel string `yaml:"loglevel"`
}

type ServerConfig struct {
	Address         string `yaml:"address"`
	Port            int    `yaml:"port"`
	ShutdownTimeout int    `yaml:"shutdownTimeout"`
	IPExtractor     string `yaml:"ipExtractor"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

type StaticConfig struct {
	Enabled bool   `yaml:"enabled"`
	Prefix  string `yaml:"prefix"`
	Root    string `yaml:"root"`
	HTML5   bool   `yaml:"html5"`
	Index   string `yaml:"index"`
	Browse  bool   `yaml:"browse"`
}

type CORSConfig struct {
	AllowOrigins     []string `yaml:"allowOrigins"`
	AllowMethods     []string `yaml:"allowMethods"`
	AllowHeaders     []string `yaml:"allowHeaders"`
	AllowCredentials bool     `yaml:"allowCredentials"`
	MaxAge           int      `yaml:"maxAge"`
}

const (
	DefaultGeneralConfigLogLevel = "info"

	DefaultServerConfigAddress         = "127.0.0.1"
	DefaultServerConfigPort            = 3000
	DefaultServerConfigShutdownTimeout = 3
	DefaultServerConfigIPExtractor     = "direct"

	DefaultDatabaseConfigHost = "localhost"
	DefaultDatabaseConfigPort = 5432
	DefaultDatabaseConfigUser = "dbuser"
	DefaultDatabaseConfigPass = "dbpass"
	DefaultDatabaseConfigName = "dbname"

	DefaultStaticConfigEnabled = false
	DefaultStaticConfigPrefix  = "/public"
	DefaultStaticConfigRoot    = "./public"
	DefaultStaticConfigHTML5   = false
	DefaultStaticConfigIndex   = "index.html"
	DefaultStaticConfigBrowse  = false

	DefaultCORSConfigAllowCredentials = false
	DefaultCORSConfigMaxAge           = 0
)

var (
	DefaultCORSConfigAllowOrigins = []string{"*"}
	DefaultCORSConfigAllowMethods = []string{"GET", "HEAD", "PUT", "PATCH", "POST", "DELETE"}
	DefaultCORSConfigAllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
)

func NewConfig(log *slog.Logger) (*Config, error) {
	c := newConfigDefaults()
	c.log = log

	configFiles := []string{
		".raptor.yaml",
		".raptor.yml",
		".raptor.conf",
		".raptor.prod.yaml",
		".raptor.prod.yml",
		".raptor.prod.conf",
		".raptor.dev.yaml",
		".raptor.dev.yml",
		".raptor.dev.conf",
	}

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

	c.ApplyEnvirontmentVariables()
	c.ApplyAppEnvironmentVariables("APP_")

	return c, nil
}

func newConfigDefaults() *Config {
	return &Config{
		GeneralConfig: GeneralConfig{
			LogLevel: DefaultGeneralConfigLogLevel,
		},
		ServerConfig: ServerConfig{
			Address:         DefaultServerConfigAddress,
			Port:            DefaultServerConfigPort,
			ShutdownTimeout: DefaultServerConfigShutdownTimeout,
			IPExtractor:     DefaultServerConfigIPExtractor,
		},
		DatabaseConfig: DatabaseConfig{
			Host:     DefaultDatabaseConfigHost,
			Port:     DefaultDatabaseConfigPort,
			Username: DefaultDatabaseConfigUser,
			Password: DefaultDatabaseConfigPass,
			Name:     DefaultDatabaseConfigName,
		},
		StaticConfig: StaticConfig{
			Enabled: DefaultStaticConfigEnabled,
			Prefix:  DefaultStaticConfigPrefix,
			Root:    DefaultStaticConfigRoot,
			HTML5:   DefaultStaticConfigHTML5,
			Index:   DefaultStaticConfigIndex,
			Browse:  DefaultStaticConfigBrowse,
		},
		CORSConfig: CORSConfig{
			AllowOrigins:     DefaultCORSConfigAllowOrigins,
			AllowMethods:     DefaultCORSConfigAllowMethods,
			AllowHeaders:     DefaultCORSConfigAllowHeaders,
			AllowCredentials: DefaultCORSConfigAllowCredentials,
			MaxAge:           DefaultCORSConfigMaxAge,
		},
		AppConfig: make(map[string]string),
	}
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

func (c *Config) ApplyEnvirontmentVariables() {
	c.ApplyEnvirontmentVariable("GENERAL_LOG_LEVEL", &c.GeneralConfig.LogLevel)

	c.ApplyEnvirontmentVariable("SERVER_ADDRESS", &c.ServerConfig.Address)
	c.ApplyEnvirontmentVariable("SERVER_PORT", &c.ServerConfig.Port)
	c.ApplyEnvirontmentVariable("SERVER_SHUTDOWN_TIMEOUT", &c.ServerConfig.ShutdownTimeout)
	c.ApplyEnvirontmentVariable("SERVER_IP_EXTRACTOR", &c.ServerConfig.IPExtractor)

	c.ApplyEnvirontmentVariable("DATABASE_HOST", &c.DatabaseConfig.Host)
	c.ApplyEnvirontmentVariable("DATABASE_PORT", &c.DatabaseConfig.Port)
	c.ApplyEnvirontmentVariable("DATABASE_USERNAME", &c.DatabaseConfig.Username)
	c.ApplyEnvirontmentVariable("DATABASE_PASSWORD", &c.DatabaseConfig.Password)
	c.ApplyEnvirontmentVariable("DATABASE_NAME", &c.DatabaseConfig.Name)

	c.ApplyEnvirontmentVariable("STATIC_ENABLED", &c.StaticConfig.Enabled)
	c.ApplyEnvirontmentVariable("STATIC_PREFIX", &c.StaticConfig.Prefix)
	c.ApplyEnvirontmentVariable("STATIC_ROOT", &c.StaticConfig.Root)
	c.ApplyEnvirontmentVariable("STATIC_HTML5", &c.StaticConfig.HTML5)
	c.ApplyEnvirontmentVariable("STATIC_INDEX", &c.StaticConfig.Index)
	c.ApplyEnvirontmentVariable("STATIC_BROWSE", &c.StaticConfig.Browse)

	c.ApplyEnvirontmentVariable("CORS_ALLOW_ORIGINS", &c.CORSConfig.AllowOrigins)
	c.ApplyEnvirontmentVariable("CORS_ALLOW_METHODS", &c.CORSConfig.AllowMethods)
	c.ApplyEnvirontmentVariable("CORS_ALLOW_HEADERS", &c.CORSConfig.AllowHeaders)
	c.ApplyEnvirontmentVariable("CORS_ALLOW_CREDENTIALS", &c.CORSConfig.AllowCredentials)
	c.ApplyEnvirontmentVariable("CORS_MAX_AGE", &c.CORSConfig.MaxAge)
}

func (c *Config) ApplyEnvirontmentVariable(key string, value interface{}) {
	if env, ok := os.LookupEnv(key); ok {
		c.log.Info("Applying environment variable", "key", key, "value", env)
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

func (c *Config) ApplyAppEnvironmentVariables(prefix string) {
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, prefix) {
			continue
		}
		key, value, found := strings.Cut(kv, "=")
		if !found {
			continue
		}
		c.log.Info("Applying app environment variable", "key", key, "value", value)
		key = strings.ToLower(strings.TrimPrefix(key, prefix))
		c.AppConfig[key] = value
	}
}
