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
