package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Station struct {
	ID   string `mapstructure:"id"`
	IP   string `mapstructure:"ip"`
	Port string `mapstructure:"port"`
}

type Config struct {
	Stations []Station `mapstructure:"stations"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config") // yaml 檔名稱 (config.yaml)
	viper.SetConfigType("yaml")   //
	viper.AddConfigPath(".")      // 從root 找file

	// viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %w", err)
	}

	return &config, nil
}
