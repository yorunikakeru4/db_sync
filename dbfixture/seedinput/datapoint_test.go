package seedinput

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"
)

func TestDatapointCountYAMLPresence(t *testing.T) {
	t.Parallel()

	var unset Datapoint
	require.NoError(t, yaml.Unmarshal([]byte("{}"), &unset))
	require.Nil(t, unset.Count)

	var zero Datapoint
	require.NoError(t, yaml.Unmarshal([]byte("count: 0"), &zero))
	require.NotNil(t, zero.Count)
	require.Equal(t, uint64(0), *zero.Count)
}
