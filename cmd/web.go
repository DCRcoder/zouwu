package main

import (
	"time"

	"github.com/DCRcoder/zouwu"
)

func main() {
	e := zouwu.NewServer(&zouwu.ServerConfig{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		Timeout:      5 * time.Second,
	})
	e.Start()
}
