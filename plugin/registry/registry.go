package registry

import (
	"xway/plugin"
)

func GetRegistry() *plugin.Registry {
	r := plugin.New()
	return r
}
