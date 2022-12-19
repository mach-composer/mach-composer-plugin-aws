package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

func TestValidateRemoteState(t *testing.T) {
	s := getSchema()
	data := map[string]any{
		"bucket":     "something",
		"key_prefix": "my-prefix",
		"region":     "eu-central-1",
	}

	schema := gojsonschema.NewRawLoader(s.RemoteStateSchema)
	document := gojsonschema.NewRawLoader(data)

	result, err := gojsonschema.Validate(schema, document)
	require.NoError(t, err)
	assert.True(t, result.Valid())
	assert.Empty(t, result.Errors())
}

func TestValidateSiteConfig(t *testing.T) {
	s := getSchema()
	data := map[string]any{
		"account_id": "12345",
		"region":     "eu-central-1",
		"default_tags": map[string]any{
			"site": "my-site",
		},
	}

	schema := gojsonschema.NewRawLoader(s.SiteConfigSchema)
	document := gojsonschema.NewRawLoader(data)

	result, err := gojsonschema.Validate(schema, document)
	require.NoError(t, err)
	assert.True(t, result.Valid())
	assert.Empty(t, result.Errors())
}

func TestValidateSiteEndpointConfig(t *testing.T) {
	s := getSchema()
	data := map[string]any{
		"throttling_burst_limit": 12345,
	}

	err := s.Validate(s.ComponentEndpointsConfigSchema, data)
	require.NoError(t, err)
}
