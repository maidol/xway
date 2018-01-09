package main

import (
	"log"
	"runtime"
	"xway/plugin/registry"
	"xway/service"
)

func main() {
	runtime.GOMAXPROCS(4)
	err := service.Run(registry.GetRegistry())
	if err != nil {
		log.Fatal(err)
	}
}
