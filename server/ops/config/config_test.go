package config

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDecodeConfig(t *testing.T) {
	testCases := []struct {
		name      string
		yaml      string
		expConfig Config
		expError  error
	}{
		{
			name:     "empty",
			expError: io.EOF,
		},
		{
			name: "single group",
			yaml: `
groups:
  - name: "exchange"
    selectors:
      - name: "*"
        type: "*"
`,
			expConfig: Config{Groups: []Group{
				{Name: "exchange", Selectors: []Selector{
					{Name: "*", Type: "*"},
				}},
			}},
		},
		{
			name: "unknown field",
			yaml: `
notgroups:
  - name: "exchange"
`,
			expError: &yaml.TypeError{Errors: []string{"line 2: field notgroups not found in type config.Config"}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := decodeConfig([]byte(tc.yaml))
			require.Equal(t, tc.expError, err)
			assert.Equal(t, tc.expConfig, c)
		})
	}
}

func TestMatchWildcard(t *testing.T) {
	testCases := []struct {
		name     string
		s        string
		match    string
		expMatch bool
	}{
		{name: "empty doesn't match exact", s: "", match: "one", expMatch: false},
		{name: "empty doesn't match partial wildcard", s: "", match: "one*", expMatch: false},
		{name: "empty matches empty", s: "", match: "", expMatch: true},
		{name: "empty matches *", s: "", match: "*", expMatch: true},
		{name: "exact match", s: "one", match: "one", expMatch: true},
		{name: "exact match on partial wildcard", s: "one", match: "one*", expMatch: true},
		{name: "exact match on multiple wildcard", s: "one", match: "one********", expMatch: true},
		{name: "exact non-match", s: "two", match: "one", expMatch: false},
		{name: "prefix match", s: "helloworld", match: "hell*", expMatch: true},
		{name: "prefix non-match", s: "helpme", match: "hell*", expMatch: false},
		{name: "middle match", s: "onetwothree", match: "*two*", expMatch: true},
		{name: "middle match with many wildcards", s: "onetwothree", match: "******two******", expMatch: true},
		{name: "multiple middle match", s: "onetwothreefourfive", match: "*two*four*", expMatch: true},
		{name: "partial match", s: "onetwothreeflourfive", match: "*two*four*", expMatch: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expMatch, matchWildcard(tc.s, tc.match))
		})
	}
}
