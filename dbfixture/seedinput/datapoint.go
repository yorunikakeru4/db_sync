package seedinput

import (
	"strconv"

	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/clickhouse/bfloat16"
	"github.com/uptrace/uptrace/pkg/unixtime"
)

// Datapoint represents a datapoint fixture input.
type Datapoint struct {
	Fingerprint uint64 `yaml:"fingerprint"`
	ProjectKey  string `yaml:"project_key"`

	Metric string        `yaml:"metric"`
	Time   unixtime.Nano `yaml:"time"`

	Instrument  models.MetricInstrument `yaml:"instrument"`
	Unit        string                  `yaml:"unit"`
	LibraryName string                  `yaml:"library_name"`

	Min       float64               `yaml:"min"`
	Max       float64               `yaml:"max"`
	Sum       float64               `yaml:"sum"`
	Count     *uint64               `yaml:"count"`
	Gauge     float64               `yaml:"gauge"`
	Histogram map[bfloat16.T]uint64 `yaml:"histogram"`

	AttrKeys   []string `yaml:"attr_keys"`
	AttrValues []string `yaml:"attr_values"`
}

// FixtureKey returns the fingerprint as a string key.
func (d *Datapoint) FixtureKey() string {
	return strconv.FormatUint(d.Fingerprint, 10)
}
