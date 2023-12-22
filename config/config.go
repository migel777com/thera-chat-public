package config

import (
	"encoding/json"
	"io"
	"log"
	"os"
)

type Config struct {
	Host              string `json:"host"`
	Port              int    `json:"port"`
	DbHost            string `json:"dbHost"`
	DbUser            string `json:"dbUser"`
	DbName            string `json:"dbName"`
	DbPass            string `json:"dbPass"`
	DbPort            int    `json:"dbPort"`
	DbMode            string `json:"dbMode"`
	DbLogMode         bool   `json:"dbLogMode"`
	CacheHost         string `json:"cacheHost"`
	CachePass         string `json:"cachePass"`
	SecretKeyAccess   string `json:"secretKeyAccess"`
	SecretKeyRefresh  string `json:"secretKeyRefresh"`
	OpenAiAuthToken   string `json:"openAiAuthToken"`
	OpenAiAssistantId string `json:"openAiAssistantId"`
}

func NewConfiguration() *Config {
	//var cfg config
	conf, err := os.Open("./config.json")
	if err != nil {
		log.Fatal(err)
		return nil
	}
	defer conf.Close()

	byteValue, _ := io.ReadAll(conf)

	var config Config
	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	return &config
}
