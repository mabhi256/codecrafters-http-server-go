package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const CRLF = "\r\n"

type HTTPRequest struct {
	Method   string
	Path     string
	Protocol string // Is usually HTTP/1.1, but keep it for the sake of completeness
	Headers  map[string]string
}

type HTTPResponse struct {
	Status  HTTPStatus
	Headers map[string]string
	Body    string
}

type HTTPStatus int

const (
	StatusOK HTTPStatus = 200
	// StatusCreated   HTTPStatus = 201
	// StatusAccepted  HTTPStatus = 202
	// StatusNoContent HTTPStatus = 204

	StatusBadRequest HTTPStatus = 400
	// StatusUnauthorized HTTPStatus = 401
	StatusForbidden HTTPStatus = 403
	StatusNotFound  HTTPStatus = 404

	StatusInternalServerError HTTPStatus = 500
)

func (s HTTPStatus) String() string {
	switch s {
	case StatusOK:
		return "200 OK"
	case StatusBadRequest:
		return "400 Bad Request"
	case StatusForbidden:
		return "403 Forbidden"
	case StatusNotFound:
		return "404 Not Found"
	case StatusInternalServerError:
		return "500 Internal Server Error"
	default:
		return fmt.Sprintf("Status %d", int(s))
	}
}

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer l.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			continue
		}

		go handleRequest(c)
	}
}

func handleRequest(c net.Conn) {
	defer c.Close()

	req, err := parseRequest(c)
	if err != nil {
		response := &HTTPResponse{
			Status:  StatusBadRequest,
			Headers: make(map[string]string),
		}

		sendResponse(c, response)
		return
	}

	response := req.RouteRequest()
	sendResponse(c, response)
}

func parseRequest(conn net.Conn) (*HTTPRequest, error) {
	reader := bufio.NewReader(conn)

	// Read request requestLine
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	// Parse request line
	parts := strings.Fields(requestLine)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid request line")
	}

	request := &HTTPRequest{
		Method:   parts[0],
		Path:     parts[1],
		Protocol: parts[2],
		Headers:  make(map[string]string),
	}

	// Parse headers
	for {
		lineRaw, err := reader.ReadString('\n')
		line := strings.TrimSpace(lineRaw)
		if err != nil || line == "" {
			break
		}

		headerParts := strings.Split(line, ":")
		if len(headerParts) == 2 {
			key := strings.ToLower(headerParts[0])
			value := strings.TrimSpace(headerParts[1])
			request.Headers[key] = value
		}
	}

	return request, nil
}

func sendResponse(conn net.Conn, res *HTTPResponse) {
	// Status Line
	response := fmt.Sprintf("HTTP/1.1 %s%s", res.Status, CRLF)

	// Headers
	res.Headers["content-length"] = strconv.Itoa(len(res.Body))
	for key, value := range res.Headers {
		response += fmt.Sprintf("%s: %s%s", key, value, CRLF)
	}

	// Body
	response += CRLF + res.Body

	conn.Write([]byte(response))
}

func (req *HTTPRequest) RouteRequest() *HTTPResponse {
	path := strings.Trim(req.Path, "/") // Remove leading and trailing '/'
	pathParts := strings.Split(path, "/")

	switch {
	case req.Path == "/":
		return &HTTPResponse{
			Status:  StatusOK,
			Headers: make(map[string]string),
		}
	case pathParts[0] == "echo" && len(pathParts) == 2:
		return &HTTPResponse{
			Status:  StatusOK,
			Headers: map[string]string{"content-type": "text/plain"},
			Body:    pathParts[1],
		}
	case pathParts[0] == "user-agent" && len(pathParts) == 1:
		return &HTTPResponse{
			Status:  StatusOK,
			Headers: map[string]string{"content-type": "text/plain"},
			Body:    req.Headers["user-agent"],
		}
	case pathParts[0] == "files" && len(pathParts) == 2:
		return handleFileRequest(pathParts[1])
	default:
		return &HTTPResponse{
			Status:  StatusNotFound,
			Headers: make(map[string]string),
		}
	}
}

func handleFileRequest(fileName string) *HTTPResponse {
	// Read the directory argument used for ./your_program.sh
	args := os.Args
	if len(args) < 2 {
		return &HTTPResponse{
			Status:  StatusInternalServerError,
			Headers: make(map[string]string),
		}
	}
	directory := args[2]

	// Read the whole file and close
	content, err := os.ReadFile(directory + fileName)
	if err != nil {
		return &HTTPResponse{
			Status:  StatusNotFound,
			Headers: make(map[string]string),
		}
	}

	return &HTTPResponse{
		Status:  StatusOK,
		Headers: map[string]string{"content-type": "application/octet-stream"},
		Body:    string(content),
	}
}
