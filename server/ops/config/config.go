package config

import (
	"flag"
	"github.com/luno/gridlock/api"
	"gopkg.in/yaml.v3"
	"os"
)

var configFile = flag.String("config", "", "path to a config yaml")

type Config struct {
	Groups []Group `yaml:"groups"`
}

type Group struct {
	Name      string     `yaml:"name"`
	Selectors []Selector `yaml:"selectors"`
}

func (g Group) MatchNode(name string, typ api.NodeType) bool {
	for _, s := range g.Selectors {
		if s.MatchNode(name, typ) {
			return true
		}
	}
	return false
}

type Selector struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

func (s Selector) MatchNode(name string, typ api.NodeType) bool {
	if s.Name != "*" && s.Name != "" && s.Name != name {
		return false
	}
	if s.Type != "*" && s.Type != "" && s.Type != string(typ) {
		return false
	}
	return true
}

var config = Config{}

func MustLoadConfig() {
	if *configFile == "" {
		return
	}
	c, err := os.ReadFile(*configFile)
	if err != nil {
		panic(err)
	}
	config, err = decodeConfig(c)
	if err != nil {
		panic(err)
	}
}

func GetConfig() Config {
	return config
}

func decodeConfig(content []byte) (Config, error) {
	var c Config
	err := yaml.Unmarshal(content, &c)
	return c, err
}
