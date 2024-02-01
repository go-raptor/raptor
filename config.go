package raptor

import (
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
)

type Config struct {
	utils *Utils

	General    General
	Server     Server
	Database   Database
	Templating Templating
	Static     Static
	CORS       CORS
}

type General struct {
	Development bool
}

type Server struct {
	Address         string
	Port            int
	ShutdownTimeout int
}

type Database struct {
	Type     string
	Host     string
	Port     int
	Username string
	Password string
	Name     string
}

type Templating struct {
	Enabled bool
	Reload  bool
}

type Static struct {
	Enabled bool
	Prefix  string
	Root    string
}

type CORS struct {
	Origins     []string
	Credentials bool
}

const (
	DefaultGeneralDevelopment = false

	DefaultServerAddress   = "127.0.0.1"
	DefaultServerPort      = 3000
	DefaultShutdownTimeout = 3

	DefaultDatabaseType = "postgres"
	DefaultDatabaseHost = "localhost"
	DefaultDatabasePort = 5432
	DefaultDatabaseUser = "app"
	DefaultDatabasePass = ""
	DefaultDatabaseName = "app"

	DefaultTemplatingEnabled = true
	DefaultTemplatingReload  = true

	DefaultStaticEnabled = true
	DefaultStaticPrefix  = "/public"
	DefaultStaticRoot    = "./public"

	DefaultCORSOrigins     = "*"
	DefaultCORSCredentials = false
)

func newConfig(u *Utils) *Config {
	c := newConfigDefaults()
	c.utils = u
	if err := c.loadConfigFromFile(".raptor.toml"); err != nil {
		c.utils.Log.Warn("Unable to load configuration file, loaded defaults...")
	}
	c.applyEnvirontmentVariables()

	return c
}

func newConfigDefaults() *Config {
	return &Config{
		General: General{
			Development: DefaultGeneralDevelopment,
		},
		Server: Server{
			Address:         DefaultServerAddress,
			Port:            DefaultServerPort,
			ShutdownTimeout: DefaultShutdownTimeout,
		},
		Database: Database{
			Type:     DefaultDatabaseType,
			Host:     DefaultDatabaseHost,
			Port:     DefaultDatabasePort,
			Username: DefaultDatabaseUser,
			Password: DefaultDatabasePass,
			Name:     DefaultDatabaseName,
		},
		Templating: Templating{
			Enabled: DefaultTemplatingEnabled,
			Reload:  DefaultTemplatingReload,
		},
		Static: Static{
			Enabled: DefaultStaticEnabled,
			Prefix:  DefaultStaticPrefix,
			Root:    DefaultStaticRoot,
		},
		CORS: CORS{
			Origins:     []string{DefaultCORSOrigins},
			Credentials: DefaultCORSCredentials,
		},
	}
}

func (c *Config) loadConfigFromFile(path string) error {
	if _, err := toml.DecodeFile(path, c); err != nil {
		return err
	}

	return nil
}

func (c *Config) applyEnvirontmentVariables() {
	c.applyEnvirontmentVariable("RAPTOR_DEVELOPMENT", &c.General.Development)

	c.applyEnvirontmentVariable("SERVER_ADDRESS", &c.Server.Address)
	c.applyEnvirontmentVariable("SERVER_PORT", &c.Server.Port)
	c.applyEnvirontmentVariable("SERVER_SHUTDOWN_TIMEOUT", &c.Server.ShutdownTimeout)

	c.applyEnvirontmentVariable("DATABASE_TYPE", &c.Database.Type)
	c.applyEnvirontmentVariable("DATABASE_HOST", &c.Database.Host)
	c.applyEnvirontmentVariable("DATABASE_PORT", &c.Database.Port)
	c.applyEnvirontmentVariable("DATABASE_USERNAME", &c.Database.Username)
	c.applyEnvirontmentVariable("DATABASE_PASSWORD", &c.Database.Password)
	c.applyEnvirontmentVariable("DATABASE_NAME", &c.Database.Name)

	c.applyEnvirontmentVariable("TEMPLATING_ENABLED", &c.Templating.Enabled)
	c.applyEnvirontmentVariable("TEMPLATING_RELOAD", &c.Templating.Reload)

	c.applyEnvirontmentVariable("STATIC_ENABLED", &c.Static.Enabled)
	c.applyEnvirontmentVariable("STATIC_PREFIX", &c.Static.Prefix)
	c.applyEnvirontmentVariable("STATIC_ROOT", &c.Static.Root)

	c.applyEnvirontmentVariable("CORS_ORIGINS", &c.CORS.Origins)
	c.applyEnvirontmentVariable("CORS_CREDENTIALS", &c.CORS.Credentials)
}

func (c *Config) applyEnvirontmentVariable(key string, value interface{}) {
	if env, ok := os.LookupEnv(key); ok {
		c.utils.Log.Info("Applying environment variable", key, env)
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
