package schema

import (
	"errors"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config object
type Config struct {
	Name         string
	IndexServer  string `toml:"index_server"`
	SearchServer string `toml:"search_server"`
	Fields       map[string]Field
}

type Setting struct {
	Schema *Schema
	Logger *Schema
	Conf   *Config
}

// LoadConf from file
func LoadConf(file string) (*Setting, error) {
	cfg := new(Config)
	_, err := toml.DecodeFile(file, cfg)
	if err != nil {
		return nil, err
	}
	return cfg.checkValid()
}

func (config *Config) checkValid() (*Setting, error) {
	if config.Name == "" {
		return nil, errors.New("Missing the name of project")
	}
	setting := &Setting{}

	if config.IndexServer == "" {
		config.IndexServer = "127.0.0.1:8383"
	}
	if config.SearchServer == "" {
		config.SearchServer = "127.0.0.1:8384"
	}
	if strings.Index(config.IndexServer, ":") == 0 {
		config.IndexServer = "127.0.0.1" + config.IndexServer
	}
	if strings.Index(config.SearchServer, ":") == 0 {
		config.SearchServer = "127.0.0.1" + config.SearchServer
	}
	if m, _ := regexp.Match("^[1-9]\\d{1,4}$", []byte(config.IndexServer)); m {
		config.IndexServer = "127.0.0.1:" + config.IndexServer
	}
	if m, _ := regexp.Match("^[1-9]\\d{1,4}$", []byte(config.SearchServer)); m {
		config.IndexServer = "127.0.0.1:" + config.SearchServer
	}
	setting.Conf = config
	sch, err := newSchema(config.Fields)
	if err != nil {
		return nil, err
	}
	setting.Schema = sch

	logCfg := new(Config)
	_, err = toml.Decode(logger, logCfg)
	if err != nil {
		return nil, err
	}
	lsch, err1 := newSchema(logCfg.Fields)
	if err1 != nil {
		return nil, err1
	}
	setting.Logger = lsch
	return setting, nil
}
