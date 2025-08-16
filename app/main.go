package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const CRLF = "\r\n"

type HTTPRequest struct {
	Method   HTTPMethod
	Path     string
	Protocol string // Is usually HTTP/1.1, but keep it for the sake of completeness
	Headers  map[string]string
	Body     []byte
}

type HTTPMethod string

const (
	HTTPGet    HTTPMethod = "GET"
	HTTPPost   HTTPMethod = "POST"
	HTTPPut    HTTPMethod = "PUT"
	HTTPDelete HTTPMethod = "DELETE"
)

type HTTPResponse struct {
	Status  HTTPStatus
	Headers map[string]string
	Body    string
}

type HTTPStatus int

const (
	StatusOK      HTTPStatus = 200
	StatusCreated HTTPStatus = 201
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
	case StatusCreated:
		return "201 Created"
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

	for {
		c.SetReadDeadline(time.Now().Add(30 * time.Second))

		req, err := parseRequest(c)

		if err != nil {
			response := &HTTPResponse{
				Status:  StatusBadRequest,
				Headers: make(map[string]string),
			}

			sendResponse(c, response)
			return
		}

		// Check if client wants to close connection
		connectionHeader := strings.ToLower(req.Headers["connection"])
		shouldClose := connectionHeader == "close"

		response := req.RouteRequest()
		if shouldClose {
			response.Headers["connection"] = "close"
		} else {
			response.Headers["connection"] = "keep-alive"
		}

		sendResponse(c, response)

		if shouldClose {
			return
		}
	}

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
		Method:   HTTPMethod(parts[0]),
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

	// Read request body
	contentLength, _ := strconv.Atoi(request.Headers["content-length"])
	if contentLength > 0 {
		bodyBytes := make([]byte, contentLength)
		_, err := io.ReadFull(reader, bodyBytes)
		if err != nil {
			return nil, err
		}
		request.Body = bodyBytes
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
	endpoint := pathParts[0]

	var pathParam string
	if len(pathParts) >= 2 {
		pathParam = pathParts[1]
	}

	switch {
	case req.Method == HTTPGet && req.Path == "/":
		return &HTTPResponse{
			Status:  StatusOK,
			Headers: make(map[string]string),
		}

	case req.Method == HTTPGet && endpoint == "echo" && len(pathParts) == 2:
		encodings, exists := req.Headers["accept-encoding"]
		if !exists {
			encodings = ""
		}
		return handleEcho(encodings, pathParam)

	case req.Method == HTTPGet && endpoint == "user-agent" && len(pathParts) == 1:
		return &HTTPResponse{
			Status:  StatusOK,
			Headers: map[string]string{"content-type": "text/plain"},
			Body:    req.Headers["user-agent"],
		}

	case req.Method == HTTPGet && endpoint == "files" && len(pathParts) == 2:
		return handleFileGet(pathParam)

	case req.Method == HTTPPost && endpoint == "files" && len(pathParts) == 2:
		return handleFilePost(pathParam, req.Body)

	default:
		return &HTTPResponse{
			Status:  StatusNotFound,
			Headers: make(map[string]string),
		}
	}
}

func handleEcho(encodings, pathParam string) *HTTPResponse {
	if strings.Contains(encodings, "gzip") {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, err := gz.Write([]byte(pathParam))
		if err != nil {
			return &HTTPResponse{
				Status:  StatusInternalServerError,
				Headers: make(map[string]string),
			}
		}

		// Close the gzip writer to finalize the stream
		err = gz.Close()
		if err != nil {
			return &HTTPResponse{
				Status:  StatusInternalServerError,
				Headers: make(map[string]string),
			}
		}

		return &HTTPResponse{
			Status:  StatusOK,
			Headers: map[string]string{"content-type": "text/plain", "content-encoding": "gzip"},
			Body:    buf.String(),
		}
	} else {
		return &HTTPResponse{
			Status:  StatusOK,
			Headers: map[string]string{"content-type": "text/plain"},
			Body:    pathParam,
		}
	}
}

func getDirectory() (string, error) {
	// Read the directory argument used for ./your_program.sh
	args := os.Args
	if len(args) < 2 {
		return "", fmt.Errorf("unable to locate directory")
	}
	return args[2], nil
}

func handleFileGet(fileName string) *HTTPResponse {
	directory, err := getDirectory()
	if err != nil {
		directory = "/tmp/"
	}

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

func handleFilePost(filename string, body []byte) *HTTPResponse {
	directory, err := getDirectory()
	if err != nil {
		directory = "/tmp/"
	}

	writeError := os.WriteFile(directory+filename, body, 0644)
	if writeError != nil {
		return &HTTPResponse{
			Status:  StatusInternalServerError,
			Headers: make(map[string]string),
		}
	}

	return &HTTPResponse{
		Status:  StatusCreated,
		Headers: make(map[string]string),
	}
}
