package engine

type Engine interface {
	GetSnapshot() (*Snapshot, error)

	Subscribe(events chan interface{}, afterIdx uint64, cancel chan struct{}) error
}
