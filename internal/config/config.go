// Package config provides functionality to load and manage
// application configuration from YAML files and environment variables.
package config

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

// Config holds the entire application configuration,
// including HTTP server, database, broker, and cache settings.
type Config struct {
	HTTPServer HTTPServer `yaml:"HTTPServer"`
	DB         DataBase   `yaml:"DataBase"`
	Broker     Broker     `yaml:"Broker"`
	Cache      Cache      `yaml:"Cache"`
}

// HTTPServer contains configuration details for the HTTP server.
type HTTPServer struct {
	Address string `yaml:"address" env-default:"localhost:8080"`
}

// DataBase contains configuration information for connecting to the database.
type DataBase struct {
	DbName   string `yaml:"db_name"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Broker contains configuration for the message broker.
type Broker struct {
	Hosts   []string `yaml:"hosts"`
	GroupID string   `yaml:"group_id"`
	Topic   string   `yaml:"topic"`
}

// Cache contains configuration for the cache, including its capacity.
type Cache struct {
	Capacity int `yaml:"capacity"`
}

// MustLoad loads the configuration from the .env file
// and a config file specified by the CONFIG_PATH environment variable,
// and returns the parsed Config. The function terminates the program on errors.
func MustLoad() *Config {
	const fn = "MustLoad"

	// Загрузить переменные окружения из `.env` файла
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("(%s) | cannot load .env file: %s", fn, err)
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatalf("(%s) | CONFIG_PATH is not set!", fn)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("(%s) | config file does not exist: %s", fn, configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("(%s) | cannot read config: %s", fn, err)
	}

	return &cfg
}
