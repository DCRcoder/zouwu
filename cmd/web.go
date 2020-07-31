package main

import (
	"fmt"
	"time"

	"github.com/DCRcoder/zouwu"
)

func main() {
	e := zouwu.NewServer(&zouwu.ServerConfig{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		Timeout:      5 * time.Second,
		Addr:         "127.0.0.1:8888",
	})
	e.GET("/hello", func(ctx *zouwu.Context) { fmt.Println("aaaa") })
	e.Start()
}
