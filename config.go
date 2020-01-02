package gott

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

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
		log.Println("Error opening config file:", err)
		log.Println("Creating default config.yml file")
		if err = ioutil.WriteFile("config.yml", []byte(defaultConfigContent), 0664); err != nil {
			log.Fatalln("Error creating default config.yml file:", err)
		}
	}

	cfg, err := config.ParseYamlBytes(file)
	if err != nil {
		log.Fatalln(fmt.Sprintf("Error parsing config file: %v", err))
	}

	if plugins, err := cfg.List("plugins"); err != nil {
		if strings.Contains(fmt.Sprintf("%s", err), "Type mismatch") ||
			strings.Contains(fmt.Sprintf("%s", err), "Invalid type") {
			log.Println("Plugins skipped: plugins property is either empty or of incorrect type. Expecting array/list of plugin names.")
		} else {
			log.Println("Couldn't parse plugins property:", err)
		}
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
