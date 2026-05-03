package dbfixture

import "embed"

// presetsFS holds built-in fixture presets loaded via WithPresets.
// Each preset is a YAML file with the same shape as SeedData.
//
//go:embed presets/*.yml
var presetsFS embed.FS
