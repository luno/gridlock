package config

import (
	"bytes"
	"flag"
	"os"
	"strings"

	"github.com/luno/gridlock/api"
	"gopkg.in/yaml.v3"
)

var configFile = flag.String("config", "", "path to a config yaml")

type Config struct {
	Groups []Group `yaml:"groups"`
}

type Group struct {
	Name      string     `yaml:"name"`
	Selectors []Selector `yaml:"selectors"`
}

func NodeMatcher(name string, typ api.NodeType) Group {
	return Group{
		Name: name,
		Selectors: []Selector{
			{Name: name, Type: string(typ)},
		},
	}
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
	Name   string `yaml:"name"`
	Prefix string `yaml:"prefix"`
	Type   string `yaml:"type"`
}

func matchWildcard(s string, match string) bool {
	if match == "" {
		return true
	}
	for i, sub := range strings.Split(match, "*") {
		if i == 0 && !strings.HasPrefix(s, sub) {
			return false
		}
		mIdx := strings.Index(s, sub)
		if mIdx == -1 {
			return false
		}
		s = s[mIdx+len(sub):]
	}
	if len(s) == 0 || match[len(match)-1] == '*' {
		return true
	}
	return false
}

func (s Selector) MatchNode(name string, typ api.NodeType) bool {
	if !strings.HasPrefix(name, s.Prefix) {
		return false
	}
	if !matchWildcard(name, s.Name) {
		return false
	}
	if !matchWildcard(string(typ), s.Type) {
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
	d := yaml.NewDecoder(bytes.NewReader(content))
	d.KnownFields(true)
	err := d.Decode(&c)
	return c, err
}
