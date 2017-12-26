package main

import (
	"log"
	"runtime"
	"xway/service"
)

func main() {
	runtime.GOMAXPROCS(4)
	err := service.Run()
	if err != nil {
		log.Fatal(err)
	}
}
