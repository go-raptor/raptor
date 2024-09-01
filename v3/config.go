package raptor

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
)

type Config struct {
	log *slog.Logger

	GeneralConfig    GeneralConfig          `toml:"General"`
	ServerConfig     ServerConfig           `toml:"Server"`
	DatabaseConfig   DatabaseConfig         `toml:"Database"`
	TemplatingConfig TemplatingConfig       `toml:"Templating"`
	StaticConfig     StaticConfig           `toml:"Static"`
	CORSConfig       CORSConfig             `toml:"CORS"`
	AppConfig        map[string]interface{} `toml:"App"`
}

type GeneralConfig struct {
	Development bool
}

type ServerConfig struct {
	Address         string
	Port            int
	ShutdownTimeout int
	ProxyHeader     string
}

type DatabaseConfig struct {
	Type     string
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
}

type CORSConfig struct {
	Origins     []string
	Credentials bool
}

const (
	DefaultGeneralConfigDevelopment = false

	DefaultServerConfigAddress = "127.0.0.1"
	DefaultServerConfigPort    = 3000
	DefaultShutdownTimeout     = 3
	DefaultProxyHeader         = ""

	DefaultDatabaseConfigType = "none"
	DefaultDatabaseConfigHost = "localhost"
	DefaultDatabaseConfigPort = 5432
	DefaultDatabaseConfigUser = "AppConfig"
	DefaultDatabaseConfigPass = ""
	DefaultDatabaseConfigName = "AppConfig"

	DefaultTemplatingConfigEnabled = false

	DefaultStaticConfigEnabled = true
	DefaultStaticConfigPrefix  = "/public"
	DefaultStaticConfigRoot    = "./public"

	DefaultCORSConfigOrigins     = "*"
	DefaultCORSConfigCredentials = false
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
			ShutdownTimeout: DefaultShutdownTimeout,
			ProxyHeader:     DefaultProxyHeader,
		},
		DatabaseConfig: DatabaseConfig{
			Type:     DefaultDatabaseConfigType,
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
		},
		CORSConfig: CORSConfig{
			Origins:     []string{DefaultCORSConfigOrigins},
			Credentials: DefaultCORSConfigCredentials,
		},
		AppConfig: make(map[string]interface{}),
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
	c.ApplyEnvirontmentVariable("SERVER_PROXY_HEADER", &c.ServerConfig.ProxyHeader)

	c.ApplyEnvirontmentVariable("DATABASE_TYPE", &c.DatabaseConfig.Type)
	c.ApplyEnvirontmentVariable("DATABASE_HOST", &c.DatabaseConfig.Host)
	c.ApplyEnvirontmentVariable("DATABASE_PORT", &c.DatabaseConfig.Port)
	c.ApplyEnvirontmentVariable("DATABASE_USERNAME", &c.DatabaseConfig.Username)
	c.ApplyEnvirontmentVariable("DATABASE_PASSWORD", &c.DatabaseConfig.Password)
	c.ApplyEnvirontmentVariable("DATABASE_NAME", &c.DatabaseConfig.Name)

	c.ApplyEnvirontmentVariable("TEMPLATING_ENABLED", &c.TemplatingConfig.Enabled)

	c.ApplyEnvirontmentVariable("STATIC_ENABLED", &c.StaticConfig.Enabled)
	c.ApplyEnvirontmentVariable("STATIC_PREFIX", &c.StaticConfig.Prefix)
	c.ApplyEnvirontmentVariable("STATIC_ROOT", &c.StaticConfig.Root)

	c.ApplyEnvirontmentVariable("CORS_ORIGINS", &c.CORSConfig.Origins)
	c.ApplyEnvirontmentVariable("CORS_CREDENTIALS", &c.CORSConfig.Credentials)
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
			*v = make([]string, 1)
			(*v)[0] = env
		default:
		}
	}
}
