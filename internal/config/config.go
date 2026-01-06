package config

import (
	"fmt"
	"os"
	"encoding/json"
	"path/filepath"
)

type Config struct {
	DataBaseURL     string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

const configFileName = "/.gatorconfig.json"

func getConfigFilePath() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(home, configFileName), nil
}

func write(cfg *Config) error {
	configPath, err := getConfigFilePath()
	if err != nil {
		return err
	}
	
	jsonData, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, jsonData, 0o644)
}

func Read() (Config, error) {
	configPath, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	return cfg, nil
}

func (cfg *Config) SetUser(name string) error {
	cfg.CurrentUserName = name
	
	return write(cfg)
}