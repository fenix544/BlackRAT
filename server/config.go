package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	BufferSize int    `json:"buffer_size"`
}

func ReadConfigFile(filename string) ([]byte, error) {
	fileContent, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return fileContent, nil
}

func ParseConfig(data []byte) (*Config, error) {
	var config Config
	err := json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
