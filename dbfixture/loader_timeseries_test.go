package dbfixture

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/unixtime"
)

func TestTimeseriesLoaderDefaultsPreserveExplicitZeroCount(t *testing.T) {
	t.Parallel()

	loader := NewTimeseriesLoader(&Fixture{
		clock: unixtime.NewFakeClock(unixtime.Unix(100, 0)),
	}, LoaderParams{})

	unset := &seedinput.Datapoint{}
	require.NoError(t, loader.Defaults(context.Background(), unset, false))
	require.NotNil(t, unset.Count)
	require.Equal(t, uint64(1), *unset.Count)

	zero := uint64(0)
	explicitZero := &seedinput.Datapoint{Count: &zero}
	require.NoError(t, loader.Defaults(context.Background(), explicitZero, false))
	require.NotNil(t, explicitZero.Count)
	require.Equal(t, uint64(0), *explicitZero.Count)

	model := new(models.Datapoint)
	_, err := loader.PopulateModel(context.Background(), model, explicitZero)
	require.NoError(t, err)
	require.Equal(t, uint64(0), model.Count)
}
