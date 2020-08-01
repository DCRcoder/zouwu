package main

import (
	"fmt"

	"github.com/DCRcoder/zouwu"
)

func main() {
	e := zouwu.NewServer()
	e.GET("/hello/:name", func(ctx *zouwu.Context) error {
		name := ctx.URLParam("name")
		return ctx.String(fmt.Sprintf("hello %s", name))
	})
	e.Run(":7777")
}
