package config

import (
	"io"
	"log"
	"os"

	"gopkg.in/yaml.v2"

	"gin-app/utils/logs"
)

type Config struct {
	AuthKey string `yaml:"auth_key"`
	Server
	Database
	RedisCache `yaml:"redis_cache"`
}

var Conf Config

func LoadConfig() error {
	configFile, err := os.Open("config.yaml")
	if err != nil {
		return err
	}

	defer func(configFile *os.File) {
		err := configFile.Close()
		if err != nil {
			logs.Error(err.Error())
			log.Fatal(err.Error())
		}
	}(configFile)

	configData, err := io.ReadAll(configFile)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(configData, &Conf)

	if err != nil {
		return err
	}

	return nil
}
