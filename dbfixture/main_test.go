package dbfixture_test

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreAnyFunction("github.com/uptrace/uptrace/pkg/unixtime.init.0.func1"),
	)

	// After all tests have run `go-snaps` will sort snapshots
	snaps.Clean(m, snaps.CleanOpts{Sort: true})
}
