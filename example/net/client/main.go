package main

import (
	"fmt"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:3001")
	if err != nil {
		fmt.Printf("dial fail:%v\n", err)
		return
	}
	go handleRead(conn)
	defer conn.Close()
	var input string
	for {
		fmt.Scanln(&input)
		conn.Write([]byte(input))
		if input == "exit" {
			fmt.Println("exit")
			return
		}
	}
}

func handleRead(conn net.Conn) {
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("err:", err)
			return
		}
		fmt.Println(string(buf[:n]))
	}
}
