package gott

import (
	"io/ioutil"
	"log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"gopkg.in/yaml.v2"
)

type tlsConfig struct {
	Listen, Cert, Key string
}

func (t tlsConfig) Enabled() bool {
	return t.Listen != "" && t.Cert != "" && t.Key != ""
}

type wssConfig struct {
	Listen, Cert, Key string
}

func (t wssConfig) Enabled() bool {
	return t.Listen != "" && t.Cert != "" && t.Key != ""
}

type loggingConfig struct {
	LogLevel          string `yaml:"log_level"`
	Filename          string
	MaxSize           int  `yaml:"max_size"`
	MaxBackups        int  `yaml:"max_backups"`
	MaxAge            int  `yaml:"max_age"`
	EnableCompression bool `yaml:"enable_compression"`
	logLevel          zapcore.Level
}

type webSocketsConfig struct {
	Listen            string
	Path              string
	RejectEmptyOrigin bool `yaml:"reject_empty_origin"`
	Origins           []string
	WSS               wssConfig `yaml:"wss"`
}

// Config holds the parsed config file
type Config struct {
	ConfigPath string
	Listen     string
	Tls        tlsConfig
	WebSockets webSocketsConfig `yaml:"websockets"`
	Logging    loggingConfig
	Plugins    []string
}

func defaultConfig() Config {
	return Config{
		Listen: ":1883",
		Tls:    tlsConfig{Listen: ":8883", Cert: "", Key: ""},
		WebSockets: webSocketsConfig{
			Listen: "",
			Path:   "/ws",
		},
		Logging: loggingConfig{
			LogLevel:          "error",
			Filename:          "gott.log",
			MaxSize:           5,
			MaxBackups:        20,
			MaxAge:            30,
			EnableCompression: true,
		},
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
		file = []byte(defaultConfigContent)
	}

	if err = yaml.Unmarshal(file, c); err != nil {
		return err
	}

	switch c.Logging.LogLevel {
	case "debug":
		c.Logging.logLevel = zap.DebugLevel
	case "info":
		c.Logging.logLevel = zap.InfoLevel
	case "error":
		c.Logging.logLevel = zap.ErrorLevel
	case "fatal":
		c.Logging.logLevel = zap.FatalLevel
	default:
		c.Logging.LogLevel = "error"
		c.Logging.logLevel = zap.ErrorLevel
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
