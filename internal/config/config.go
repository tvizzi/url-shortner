package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Env         string `yaml:"env" env-default:"local"`
	DatabaseURL string `yaml:"database_url" env:"DATABASE_URL"`
	AliasLength int    `yaml:"alias_length" env:"ALIAS_LENGTH" env-default:"6"`
	HTTPServer  `yaml:"http_server"`
	Auth        `yaml:"auth"`
}

type Auth struct {
	User     string `yaml:"user" env:"AUTH_USER"`
	Password string `yaml:"password" env:"AUTH_PASSWORD"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"localhost:8080"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

func MustLoad() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on environment variables")
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.DatabaseURL = dbURL
		log.Println("Using DATABASE_URL from environment")
	}

	// 3. Явная загрузка из переменных окружения (может перезаписать значения из YAML)
	if user := os.Getenv("AUTH_USER"); user != "" {
		cfg.Auth.User = user
		log.Println("Using AUTH_USER from environment")
	}

	if password := os.Getenv("AUTH_PASSWORD"); password != "" {
		cfg.Auth.Password = password
		log.Println("Using AUTH_PASSWORD from environment")
	}

	if cfg.Auth.User == "" || cfg.Auth.Password == "" {
		log.Fatal("Username and Password is empty")
	}

	if aliasLenStr := os.Getenv("ALIAS_LENGTH"); aliasLenStr != "" {
		aliasLen, err := strconv.Atoi(aliasLenStr)
		if err != nil {
			log.Fatalf("invalid ALIAS_LENGTH value %q: %s", aliasLenStr, err)
		}
		cfg.AliasLength = aliasLen
		log.Println("Using ALIAS_LENGTH from environment")
	}

	return &cfg
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{Env:%s, Address:%s, AliasLength:%d, AuthUser:%s}",
		c.Env, c.Address, c.AliasLength, c.Auth.User)
}
