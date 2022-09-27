package main

import (
	"fmt"

	"github.com/molei20021/fastrpc"
)

func main() {
	var event fastrpc.Event
	event.LoopNum = 1
	event.Serve = func(svr fastrpc.Server) (action fastrpc.Action) {
		fmt.Printf("enter serve \n")
		return
	}
	event.Data = func(c fastrpc.Conn, in []byte) (out []byte, action fastrpc.Action) {
		fmt.Printf("enter data \n")
		out = in
		return
	}
	fastrpc.Serve(event, "localhost:3001")
}
