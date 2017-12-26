package engine

type Engine interface {
	GetSnapshot() (interface{}, error)
}
