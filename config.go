package gott

import (
	"io/ioutil"
	"log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"gopkg.in/yaml.v2"
)

type Config struct {
	ConfigPath string
	Listen     string
	LogLevel   string `yaml:"logLevel"`
	Plugins    []string
	logLevel   zapcore.Level
}

func defaultConfig() Config {
	return Config{
		Listen:     ":1883",
		LogLevel:   "error",
		ConfigPath: "config.yml",
	}
}

func (c *Config) loadConfig() error {
	file, err := ioutil.ReadFile(c.ConfigPath)
	if err != nil {
		log.Println("Error opening config file:", err)
		log.Println("Creating default config.yml file")
		if err = ioutil.WriteFile("config.yml", []byte(defaultConfigContent), 0664); err != nil {
			log.Fatalln("Error creating default config.yml file:", err)
		}
	}

	if err = yaml.Unmarshal(file, c); err != nil {
		return err
	}

	switch c.LogLevel {
	case "debug":
		c.logLevel = zap.DebugLevel
	case "info":
		c.logLevel = zap.InfoLevel
	case "error":
		c.logLevel = zap.ErrorLevel
	case "fatal":
		c.logLevel = zap.FatalLevel
	default:
		c.LogLevel = "error"
		c.logLevel = zap.ErrorLevel
	}

	return nil
}

func newConfig() (Config, error) {
	cnf := defaultConfig()
	if err := cnf.loadConfig(); err != nil {
		return cnf, err
	}
	return cnf, nil
}
