package engine

import (
	"fmt"
)

// FrontendUpserted add/update
type FrontendUpserted struct {
	Frontend Frontend
}

func (f *FrontendUpserted) String() string {
	return fmt.Sprintf("FrontendUpserted(frontend=%v)", &f.Frontend)
}
