package registry

import (
	"xway/plugin"
	"xway/plugin/authtoken"
)

func GetRegistry() *plugin.Registry {
	r := plugin.New()
	specs := []*plugin.MiddlewareSpec{
		&plugin.MiddlewareSpec{Type: "oauth", MW: authtoken.New},
	}
	for _, spec := range specs {
		if err := r.AddMW(spec); err != nil {
			panic(err)
		}
	}
	return r
}
