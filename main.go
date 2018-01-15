package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"xway/plugin/registry"
	"xway/service"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	num := runtime.NumCPU()
	fmt.Printf("[NumCPU] %v\n", num)
	gmp := os.Getenv("GOMAXPROCS")
	if gmp != "" {
		r, e := strconv.Atoi(gmp)
		if e == nil && r < num && r > 0 {
			num = r
		}
	}
	fmt.Printf("[GOMAXPROCS] %v\n", num)
	err := service.Run(registry.GetRegistry())
	if err != nil {
		log.Fatal(err)
	}
}
