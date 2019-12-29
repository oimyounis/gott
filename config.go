package gott

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/olebedev/config"
)

var defaults = map[string]interface{}{
	"ConfigPath": "config.yml",
}

type Config struct {
	ConfigPath string
	Plugins    []string
}

func (c *Config) LoadConfig() {
	file, err := ioutil.ReadFile(c.ConfigPath)
	if err != nil {
		log.Fatalln(fmt.Sprintf("Error opening config file: %v", err))
	}

	cfg, err := config.ParseYamlBytes(file)
	if err != nil {
		log.Fatalln(fmt.Sprintf("Error parsing config file: %v", err))
	}

	if plugins, err := cfg.List("plugins"); err != nil {
		log.Println("Error parsing plugins:", err)
	} else {
		for _, plugin := range plugins {
			if p, ok := plugin.(string); !ok {
				log.Println("Couldn't parse plugin:", plugin)
			} else {
				c.Plugins = append(c.Plugins, p)
			}
		}
	}
}

func NewConfig() Config {
	cnf := Config{ConfigPath: defaults["ConfigPath"].(string)}
	cnf.LoadConfig()
	return cnf
}
