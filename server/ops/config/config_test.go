package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDecodeConfig(t *testing.T) {
	testCases := []struct {
		name      string
		yaml      string
		expConfig Config
	}{
		{name: "empty"},
		{name: "single group",
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := decodeConfig([]byte(tc.yaml))
			require.NoError(t, err)
			assert.Equal(t, tc.expConfig, c)
		})
	}

}
