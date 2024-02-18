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
	Reload  bool
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

	DefaultDatabaseConfigType = "none"
	DefaultDatabaseConfigHost = "localhost"
	DefaultDatabaseConfigPort = 5432
	DefaultDatabaseConfigUser = "AppConfig"
	DefaultDatabaseConfigPass = ""
	DefaultDatabaseConfigName = "AppConfig"

	DefaultTemplatingConfigEnabled = true
	DefaultTemplatingConfigReload  = true

	DefaultStaticConfigEnabled = true
	DefaultStaticConfigPrefix  = "/public"
	DefaultStaticConfigRoot    = "./public"

	DefaultCORSConfigOrigins     = "*"
	DefaultCORSConfigCredentials = false
)

func newConfig(log *slog.Logger) *Config {
	c := newConfigDefaults()
	c.log = log

	err := c.loadConfigFromFile(".raptor.toml")
	if err != nil {
		err = c.loadConfigFromFile(".raptor.dev.toml")
	}
	if err != nil {
		log.Warn("Unable to load configuration file, loaded defaults...")
	}

	c.AppConfiglyEnvirontmentVariables()

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
			Reload:  DefaultTemplatingConfigReload,
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

func (c *Config) AppConfiglyEnvirontmentVariables() {
	c.AppConfiglyEnvirontmentVariable("RAPTOR_DEVELOPMENT", &c.GeneralConfig.Development)

	c.AppConfiglyEnvirontmentVariable("ServerConfig_ADDRESS", &c.ServerConfig.Address)
	c.AppConfiglyEnvirontmentVariable("ServerConfig_PORT", &c.ServerConfig.Port)
	c.AppConfiglyEnvirontmentVariable("ServerConfig_SHUTDOWN_TIMEOUT", &c.ServerConfig.ShutdownTimeout)

	c.AppConfiglyEnvirontmentVariable("DatabaseConfig_TYPE", &c.DatabaseConfig.Type)
	c.AppConfiglyEnvirontmentVariable("DatabaseConfig_HOST", &c.DatabaseConfig.Host)
	c.AppConfiglyEnvirontmentVariable("DatabaseConfig_PORT", &c.DatabaseConfig.Port)
	c.AppConfiglyEnvirontmentVariable("DatabaseConfig_USERNAME", &c.DatabaseConfig.Username)
	c.AppConfiglyEnvirontmentVariable("DatabaseConfig_PASSWORD", &c.DatabaseConfig.Password)
	c.AppConfiglyEnvirontmentVariable("DatabaseConfig_NAME", &c.DatabaseConfig.Name)

	c.AppConfiglyEnvirontmentVariable("TemplatingConfig_ENABLED", &c.TemplatingConfig.Enabled)
	c.AppConfiglyEnvirontmentVariable("TemplatingConfig_RELOAD", &c.TemplatingConfig.Reload)

	c.AppConfiglyEnvirontmentVariable("StaticConfig_ENABLED", &c.StaticConfig.Enabled)
	c.AppConfiglyEnvirontmentVariable("StaticConfig_PREFIX", &c.StaticConfig.Prefix)
	c.AppConfiglyEnvirontmentVariable("StaticConfig_ROOT", &c.StaticConfig.Root)

	c.AppConfiglyEnvirontmentVariable("CORSConfig_ORIGINS", &c.CORSConfig.Origins)
	c.AppConfiglyEnvirontmentVariable("CORSConfig_CREDENTIALS", &c.CORSConfig.Credentials)
}

func (c *Config) AppConfiglyEnvirontmentVariable(key string, value interface{}) {
	if env, ok := os.LookupEnv(key); ok {
		c.log.Info("AppConfiglying environment variable", key, env)
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
