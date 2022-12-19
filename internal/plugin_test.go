package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetSiteEndpointsConfig(t *testing.T) {
	plugin := NewAWSPlugin()
	err := plugin.SetSiteConfig("my-site", map[string]any{
		"account_id": "12345",
		"region":     "eu-central-1",
	})
	require.NoError(t, err)

	err = plugin.SetComponentConfig("my-component", map[string]any{
		"integrations": []string{"aws"},
	})
	require.NoError(t, err)

	err = plugin.SetComponentEndpointsConfig("my-component", map[string]string{
		"internal": "internal",
	})
	require.NoError(t, err)

	err = plugin.SetSiteEndpointConfig("my-site", "internal", map[string]any{
		"url":                    "foobar",
		"throttling_burst_limit": 5000,
		"throttling_rate_limit":  10000,
	})
	require.NoError(t, err)

	result, err := plugin.RenderTerraformResources("my-site")
	require.NoError(t, err)
	assert.Contains(t, result, "throttling_burst_limit = 5000")
}
