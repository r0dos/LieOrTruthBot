package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the options of r0bot.
type Config struct {
	BotToken  string `yaml:"bot_token"`
	SuperUser int64  `yaml:"super_user"`
}

// NewConfig returns a new decoded Config struct
func NewConfig(configPath string) (*Config, error) {
	// Create config structure
	config := &Config{}

	// Open config file
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("file open: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Init new YAML decode
	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	return config, nil
}
