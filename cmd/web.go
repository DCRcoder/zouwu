package main

import (
	"errors"

	"github.com/DCRcoder/zouwu"
)

func main() {
	e := zouwu.NewServer()
	e.GET("/hello/:id", func(ctx *zouwu.Context) error {
		return errors.New("fuck")
	})
	e.Run(":7777")
}
