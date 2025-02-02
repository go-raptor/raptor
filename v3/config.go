package raptor

import (
	"log/slog"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	log *slog.Logger

	GeneralConfig    GeneralConfig     `toml:"General"`
	ServerConfig     ServerConfig      `toml:"Server"`
	DatabaseConfig   DatabaseConfig    `toml:"Database"`
	TemplatingConfig TemplatingConfig  `toml:"Templating"`
	StaticConfig     StaticConfig      `toml:"Static"`
	CORSConfig       CORSConfig        `toml:"CORS"`
	AppConfig        map[string]string `toml:"App"`
}

type GeneralConfig struct {
	Development bool
}

type ServerConfig struct {
	Address         string
	Port            int
	ShutdownTimeout int
	IPExtractor     string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Name     string
}

type TemplatingConfig struct {
	Enabled bool
}

type StaticConfig struct {
	Enabled bool
	Prefix  string
	Root    string
	HTML5   bool
	Index   string
	Browse  bool
}

type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	MaxAge           int
}

const (
	DefaultGeneralConfigDevelopment = false

	DefaultServerConfigAddress         = "127.0.0.1"
	DefaultServerConfigPort            = 3000
	DefaultServerConfigShutdownTimeout = 3
	DefaultServerConfigIPExtractor     = "direct"

	DefaultDatabaseConfigHost = "localhost"
	DefaultDatabaseConfigPort = 5432
	DefaultDatabaseConfigUser = "dbuser"
	DefaultDatabaseConfigPass = "dbpass"
	DefaultDatabaseConfigName = "dbname"

	DefaultTemplatingConfigEnabled = false

	DefaultStaticConfigEnabled = true
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

func newConfig(log *slog.Logger) *Config {
	c := newConfigDefaults()
	c.log = log

	configFiles := []string{
		".raptor.toml",
		".raptor.conf",
		".raptor.prod.toml",
		".raptor.prod.conf",
		".raptor.dev.toml",
		".raptor.dev.conf",
	}

	var err error
	for _, file := range configFiles {
		err = c.loadConfigFromFile(file)
		if err == nil {
			break
		}
	}

	if err != nil {
		log.Warn("Unable to load configuration file, loaded defaults...")
	}

	c.ApplyEnvirontmentVariables()
	c.ApplyAppEnvironmentVariables("APP_")

	return c
}

func newConfigDefaults() *Config {
	return &Config{
		GeneralConfig: GeneralConfig{
			Development: DefaultGeneralConfigDevelopment,
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
		TemplatingConfig: TemplatingConfig{
			Enabled: DefaultTemplatingConfigEnabled,
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

func WithConfig(config *Config) RaptorOption {
	return func(r *Raptor) {
		if config != nil {
			mergeConfig(r.Utils.Config, config)
		}
	}
}

func mergeConfig(dst, src *Config) {
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

		if srcField.Kind() == reflect.Struct {
			for j := 0; j < srcField.NumField(); j++ {
				nestedSrcField := srcField.Field(j)
				nestedDstField := dstField.Field(j)

				if !nestedSrcField.IsZero() {
					nestedDstField.Set(nestedSrcField)
				}
			}
		}
	}
}

func (c *Config) loadConfigFromFile(path string) error {
	if _, err := toml.DecodeFile(path, c); err != nil {
		return err
	}

	return nil
}

func (c *Config) ApplyEnvirontmentVariables() {
	c.ApplyEnvirontmentVariable("RAPTOR_DEVELOPMENT", &c.GeneralConfig.Development)

	c.ApplyEnvirontmentVariable("SERVER_ADDRESS", &c.ServerConfig.Address)
	c.ApplyEnvirontmentVariable("SERVER_PORT", &c.ServerConfig.Port)
	c.ApplyEnvirontmentVariable("SERVER_SHUTDOWN_TIMEOUT", &c.ServerConfig.ShutdownTimeout)
	c.ApplyEnvirontmentVariable("SERVER_IP_EXTRACTOR", &c.ServerConfig.IPExtractor)

	c.ApplyEnvirontmentVariable("DATABASE_HOST", &c.DatabaseConfig.Host)
	c.ApplyEnvirontmentVariable("DATABASE_PORT", &c.DatabaseConfig.Port)
	c.ApplyEnvirontmentVariable("DATABASE_USERNAME", &c.DatabaseConfig.Username)
	c.ApplyEnvirontmentVariable("DATABASE_PASSWORD", &c.DatabaseConfig.Password)
	c.ApplyEnvirontmentVariable("DATABASE_NAME", &c.DatabaseConfig.Name)

	c.ApplyEnvirontmentVariable("TEMPLATING_ENABLED", &c.TemplatingConfig.Enabled)

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
		c.log.Info("Applying environment variable", key, env)
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
		c.log.Info("Applying environment variable", key, value)
		key = strings.ToLower(strings.TrimPrefix(key, prefix))
		c.AppConfig[key] = value
	}
}
