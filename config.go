package raptor

import (
	"log"
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
)

type Config struct {
	General    General
	Server     Server
	Templating Templating
	CORS       CORS
}

type General struct {
	Development bool
}

type Server struct {
	Address string
	Port    int
}

type Templating struct {
	Enabled bool
	Reload  bool
}

type CORS struct {
	Origins     []string
	Credentials bool
}

const (
	DefaultGeneralDevelopment = false

	DefaultServerAddress = "127.0.0.1"
	DefaultServerPort    = 3000

	DefaultTemplatingEnabled = true
	DefaultTemplatingReload  = true

	DefaultCORSOrigins     = "*"
	DefaultCORSCredentials = false
)

func NewConfig() *Config {
	c := NewConfigDefaults()
	if err := c.loadConfigFromFile(".raptor.toml"); err != nil {
		log.Println("Unable to load configuration file, loaded defaults...")
	}
	c.applyEnvirontmentVariables()

	return c
}

func NewConfigDefaults() *Config {
	return &Config{
		General: General{
			Development: DefaultGeneralDevelopment,
		},
		Server: Server{
			Address: DefaultServerAddress,
			Port:    DefaultServerPort,
		},
		Templating: Templating{
			Enabled: DefaultTemplatingEnabled,
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
	applyEnvirontmentVariable("RAPTOR_DEVELOPMENT", &c.General.Development)

	applyEnvirontmentVariable("SERVER_ADDRESS", &c.Server.Address)
	applyEnvirontmentVariable("SERVER_PORT", &c.Server.Port)

	applyEnvirontmentVariable("TEMPLATING_ENABLED", &c.Templating.Enabled)
	applyEnvirontmentVariable("TEMPLATING_RELOAD", &c.Templating.Reload)

	applyEnvirontmentVariable("CORS_ORIGINS", &c.CORS.Origins)
	applyEnvirontmentVariable("CORS_CREDENTIALS", &c.CORS.Credentials)
}

func applyEnvirontmentVariable(key string, value interface{}) {
	if env, ok := os.LookupEnv(key); ok {
		log.Println("Applying environment variable", key)
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
		}
	}
}
