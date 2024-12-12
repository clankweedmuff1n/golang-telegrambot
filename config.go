package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	TargetChatIDs []string `json:"target_chat_ids"`
}

func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("could not decode config: %w", err)
	}

	return &config, nil
}
