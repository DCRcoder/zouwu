# ZouWu（驺吾）
注: 驺吾《山海经·海内北经》：“ 林氏国 ，有珍兽，大若虎，五采毕具，尾长于身，名曰驺吾，乘之日行千里。”

ZouWu is a experimental web framework built on top of [fasthttp](https://github.com/valyala/fasthttp). 
the project base on [kratos](https://github.com/go-kratos/kratos) and [fiber](https://github.com/gofiber/fiber)


## Quickstart

```go
package main

import (
	"github.com/DCRcoder/zouwu"
)

func main() {
	e := zouwu.NewServer()
	e.GET("/hello", func(ctx *zouwu.Context) error {
		return ctx.String("hello world")
	})
	e.Run(":7777")
}
```
