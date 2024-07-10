package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

type HTTPRequest struct {
	conn    net.Conn
	dir     *string
	method  string
	path    string
	body    string
	headers map[string]string
}

type HTTPResponse struct {
	protocol      string
	body          string
	status        string
	contentType   string
	contentLength int
	statusCode    int
}

func parseRequest(r string) HTTPRequest {
	var req HTTPRequest
	head, body, _ := strings.Cut(r, "\r\n\r\n")
	headLines := strings.Split(head, "\r\n")

	hs := make(map[string]string)

	for _, h := range headLines[1:] {
		key, value, _ := strings.Cut(h, ":")
		key = strings.ToLower(key)
		value = strings.Trim(value, " \r\n")
		hs[key] = value
	}

	req.headers = hs
	url := strings.Split(headLines[0], " ")
	req.method = url[0]
	req.path = url[1]
	req.body = string(strings.Trim(body, " \r\n"))

	fmt.Printf("%+v\n", req)

	return req
}

func (res *HTTPResponse) build(body, status, contentType string, statusCode int) {
	res.protocol = "HTTP/1.1"
	res.body = body
	res.status = status
	res.contentType = contentType
	res.statusCode = statusCode
	res.contentLength = len(body)
}

func (res *HTTPResponse) make() []byte {
	res.contentLength = len(res.body)
	if res.protocol == "" {
		res.protocol = "HTTP/1.1"
	}
	s := fmt.Sprintf(
		"%s %d %s\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n%s",
		res.protocol,
		res.statusCode,
		res.status,
		res.contentType,
		res.contentLength,
		res.body,
	)
	return []byte(s)
}

func handleConnection(req HTTPRequest) {
	reqBuff := make([]byte, 1024)
	res := HTTPResponse{}
	conn := req.conn
	_, err := conn.Read(reqBuff)
	if err != nil {
		log.Fatalf("%s", err)
	}
	r := parseRequest(string(reqBuff))

	// path := strings.Split(string(reqBuff), " ")[1]

	if r.path == "/" {
		_, err = conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else if strings.HasPrefix(r.path, "/echo/") {
		res.body, _ = strings.CutPrefix(r.path, "/echo/")
		res.status = "OK"
		res.statusCode = 200
		res.contentType = "text/plain;charset=UTF-8"
		_, err = conn.Write(res.make())
	} else if strings.HasPrefix(r.path, "/user-agent") {
		res.body = r.headers["user-agent"]
		res.status = "OK"
		res.statusCode = 200
		res.contentType = "text/plain;charset=UTF-8"
		_, err = conn.Write(res.make())
	} else if strings.HasPrefix(r.path, "/files") {
		if *dir == "" {
			log.Println("server runs without file support")
			_, _ = conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			conn.Close()
			return
		}
		if _, err = os.Stat(*dir); os.IsNotExist(err) {
			log.Println("this endpoint need directory flag to be given")
			_, _ = conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			conn.Close()
			return
		}

		_, filename, _ := strings.Cut(r.path, "/files/")
		filePath := *dir + "/" + filename
		if r.method == "POST" {
			log.Printf("parsing POST request with file path: %s", filePath)
			fmt.Printf("Body: %v", r.headers["content-length"])
			bodyLength, _ := strconv.Atoi(r.headers["content-length"])
			dat := []byte(r.body)[:bodyLength]
			err = os.WriteFile(filePath, dat, 0666)
			if err != nil {
				log.Printf("error creating file: %s \n%s", filePath, err)
			}
			res.body = string(dat)
			res.statusCode = 201
			res.status = "Created"
			_, _ = conn.Write(res.make())
			return
		} else {
			dat, err := os.ReadFile(filePath)
			if err != nil {
				_, _ = conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
				conn.Close()
				return
			}
			res.body = string(dat)
			res.statusCode = 200
			res.status = "OK"
			res.contentType = "application/octet-stream"
			_, _ = conn.Write(res.make())

		}
		conn.Close()
	} else {
		_, err = conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
	if err != nil {
		log.Fatalln("Error while writing response.")
	}
	conn.Close()
}

var (
	port = 4221
	dir  *string
)

func main() {
	dir = flag.String("directory", "", "directory contains files")
	flag.Parse()
	address := fmt.Sprintf(":%d", port)
	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to bind to port %d\n", port)
	}

	defer l.Close()
	log.Printf("Starting HTTP server on port: %d\n", port)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatalln("Error accepting connection: ", err.Error())
		}
		req := HTTPRequest{
			dir:  dir,
			conn: conn,
		}
		go handleConnection(req)
	}
}
