package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	DbUrl       string            `mapstructure:"db_url"`
	Collaborate CollaborateConfig `mapstructure:"collaborate"`
	Port        int               `mapstructure:"port"`
	
	// JWT Configuration
	JWTSecret string `mapstructure:"jwt_secret"`
	JWTExpiry string `mapstructure:"jwt_expires_in"`
	
	// Admin Configuration
	Admin AdminConfig `mapstructure:"admin"`
	
	// Email Configuration
	Email EmailConfig `mapstructure:"email"`
	
	// Frontend URL
	FrontendURL string `mapstructure:"frontend_url"`
	
	// Environment
	Environment string `mapstructure:"env"`
}

type CollaborateConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type AdminConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type EmailConfig struct {
	Provider string `mapstructure:"provider"`
	APIKey   string `mapstructure:"api_key"`
	From     string `mapstructure:"from"`
	FromName string `mapstructure:"from_name"`
}

func NewConfig() (Config, error) {
	_ = godotenv.Load()
	configPath := os.Getenv("CONFIG_PATH")

	if configPath != "" {
		fmt.Println("Using config path:", configPath)
		viper.SetConfigFile(configPath)
	} else {
		fmt.Println("Using default config path: ./config.yaml")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}

	err := viper.ReadInConfig()
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		return Config{}, err
	}
	return config, nil
}
