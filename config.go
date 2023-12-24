package raptor

import (
	"log"
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server Server
}

type Server struct {
	Address string
	Port    int
}

const (
	DefaultServerAddress = "127.0.0.1"
	DefaultServerPort    = 3000
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
		Server: Server{
			Address: DefaultServerAddress,
			Port:    DefaultServerPort,
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
	applyEnvirontmentVariable("SERVER_ADDRESS", &c.Server.Address)
	applyEnvirontmentVariable("SERVER_PORT", &c.Server.Port)
}

func applyEnvirontmentVariable(key string, value interface{}) {
	if env, ok := os.LookupEnv(key); ok {
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
		}
	}
}
