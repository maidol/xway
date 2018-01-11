package main

import (
	"log"
	"runtime"
	"xway/plugin/registry"
	"xway/service"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	runtime.GOMAXPROCS(4)
	err := service.Run(registry.GetRegistry())
	if err != nil {
		log.Fatal(err)
	}
}
