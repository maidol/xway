package engine

type Engine interface {
	GetSnapshot() (*Snapshot, error)
}
