package middleware

import (
	"fmt"

	"github.com/DCRcoder/zouwu"
)

// Recover will recover from panics and calls
func Recover() zouwu.HandlerFunc {
	return func(ctx *zouwu.Context) error {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", err)
				}
				ctx.Errors(err)
				return
			}
		}()
		ctx.Next()
		return nil
	}
}
