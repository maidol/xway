package engine

import (
	"fmt"
)

// FrontendUpserted add/update event
type FrontendUpserted struct {
	Frontend Frontend
}

func (f *FrontendUpserted) String() string {
	return fmt.Sprintf("FrontendUpserted(frontend=%v)", &f.Frontend)
}

// FrontendDeleted delete event
type FrontendDeleted struct {
	FrontendKey FrontendKey
}

func (f *FrontendDeleted) String() string {
	return fmt.Sprintf("FrontendDeleted(frontendKey=%v)", &f.FrontendKey)
}

type FrontendKey struct {
	Id string
}
