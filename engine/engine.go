package engine

import "xway/plugin"

type Engine interface {
	GetSnapshot() (*Snapshot, error)

	Subscribe(events chan interface{}, afterIdx uint64, cancel chan struct{}) error

	// GetRegistry returns registry with the supported plugins. It should be stored by Engine instance.
	GetRegistry() *plugin.Registry
}
