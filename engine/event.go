package engine

import (
	"fmt"
)

type FrontendKey struct {
	Id string
}

// FrontendUpserted add/update
type FrontendUpserted struct {
	Frontend Frontend
}

func (f *FrontendUpserted) String() string {
	return fmt.Sprintf("FrontendUpserted(frontend=%v)", &f.Frontend)
}

type FrontendDeleted struct {
	FrontendKey FrontendKey
}

func (f *FrontendDeleted) String() string {
	return fmt.Sprintf("FrontendDeleted(frontendKey=%v)", &f.FrontendKey)
}
