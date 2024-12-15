package config

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	HTTPServer HTTPServer `yaml:"HTTPServer"`
	DB         DataBase   `yaml:"DataBase"`
	Broker     Broker     `yaml:"Broker"`
	Cache      Cache      `yaml:"Cache"`
}

type HTTPServer struct {
	Address string `yaml:"address" env-default:"localhost:8080"`
}

type DataBase struct {
	DbName   string `yaml:"db_name"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Broker struct {
	Hosts   []string `yaml:"hosts"`
	GroupID string   `yaml:"group_id"`
	Topic   string   `yaml:"topic"`
}

type Cache struct {
	Capacity int `yaml:"capacity"`
}

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
