package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports above (feel free to remove this!)
// var _ = net.Listen
// var _ = os.Exit

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	c, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	reader := bufio.NewReader(c)
	line, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
		os.Exit(1)
	}

	req := strings.Fields(line)
	target := req[1]

	if target == "/" {
		response := "HTTP/1.1 200 OK\r\n\r\n"
		c.Write([]byte(response))
	} else {
		response := "HTTP/1.1 404 Not Found\r\n\r\n"
		c.Write([]byte(response))
	}
}
