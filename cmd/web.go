package main

import (
	"github.com/DCRcoder/zouwu"
	"github.com/DCRcoder/zouwu/middleware"
)

func main() {
	e := zouwu.NewServer()
	e.Use(middleware.Recover())
	e.GET("/hello/:id", func(ctx *zouwu.Context) error {
		panic("gogogo")
	})
	e.Run(":7777")
}
