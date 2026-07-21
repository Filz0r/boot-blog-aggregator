package config

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
)

type Config struct {
	DbUrl    string  `json:"db_url"`
	Username *string `json:"current_user_name"`
}

func getConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(home, ".gatorconfig.json")

	return path, nil
}

func Read() (Config, error) {
	path, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, nil
	}

	res := Config{}

	err = json.Unmarshal(data, &res)
	if err != nil {
		return Config{}, err
	}

	return res, nil
}

func (cfg *Config) SetUser(username *string) error {
	name, err := GetUsername()
	if err != nil {
		return err
	}
	if username == nil {
		username = &name
	}
	cfg.Username = username
	err = write(*cfg)
	if err != nil {
		return err
	}
	return nil
}

func write(cfg Config) error {
	path, err := getConfigFilePath()
	if err != nil {
		return err
	}
	converted, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, converted, 0o600)
	if err != nil {
		return err
	}
	return nil
}

func GetUsername() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	return user.Username, nil
}
