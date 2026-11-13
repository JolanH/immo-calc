package storage

import "go.yaml.in/yaml/v3"

// CurrentVersion is the schema_version this build reads and writes.
const CurrentVersion = 1

// Migration upgrades a document's sections from version N to N+1, in place.
type Migration func(sections map[string]*yaml.Node) error

// migrations maps a version N to the function upgrading it to N+1.
// v1 is the first version, so there are none yet.
var migrations = map[int]Migration{}
