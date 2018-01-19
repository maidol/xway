package clientlogstatus

import (
	"fmt"
	"net/http"
	"xway/plugin/handler"

	"github.com/urfave/negroni"
)

type ClientLogStatus struct {
	handler.StdErrorHandler
	opt Options
}

type Options struct {
}

func New(opt interface{}) negroni.Handler {
	o, ok := opt.(Options)
	if !ok {
		o = Options{}
	}
	return &ClientLogStatus{
		opt: o,
	}
}

func (cls *ClientLogStatus) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	fmt.Println("[mw:clientlogstatus]")
	next(rw, r)
}
