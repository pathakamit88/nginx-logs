package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Addresses []string `yaml:"addresses"`
	Username  string   `yaml:"username"`
	Password  string   `yaml:"password"`
}

func ReadConfig(path string) (*Config, error) {
	//home, err := os.UserHomeDir()
	//if err != nil {
	//	return nil, err
	//}
	//path := filepath.Join(home, "opensearch_config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
