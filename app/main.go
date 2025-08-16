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

const CLRF = "\r\n"

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	OKStatus := Status("200 OK")
	NotFoundStatus := Status("404 Not Found")
	PlainTextContent := ContentType("text/plain")

	// for {
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

	paths := strings.Split(target, "/")

	switch {
	case target == "/":
		response := OKStatus + ContentLength(0) + CLRF
		c.Write([]byte(response))
	case paths[0] == "" && paths[1] == "echo" && len(paths) == 3:
		pathParam := paths[2]
		response := OKStatus +
			PlainTextContent +
			ContentLength(len(pathParam)) +
			CLRF + pathParam
		c.Write([]byte(response))
	default:
		response := NotFoundStatus + ContentLength(0) + CLRF
		c.Write([]byte(response))
	}

	// 	c.Close()
	// }
}

func Status(status string) string {
	return fmt.Sprintf("HTTP/1.1 %s%s", status, CLRF)
}

func ContentType(ctype string) string {
	return fmt.Sprintf("Content-Type: %s%s", ctype, CLRF)
}

func ContentLength(length int) string {
	return fmt.Sprintf("Content-Length: %v%s", length, CLRF)
}
