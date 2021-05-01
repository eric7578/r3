package main

import (
	"context"
	"time"

	"github.com/eric7578/r3"
)

func main() {
	d := r3.Daemon{
		RendererAwake: time.Second * 30,
	}
	d.Run(context.Background(), ":8080")
}
