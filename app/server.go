package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

type HTTPRequest struct {
	conn net.Conn
	dir  *string
}

type HTTPResponse struct {
	protocol      string
	body          string
	status        string
	contentType   string
	contentLength int
	statusCode    int
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

	path := strings.Split(string(reqBuff), " ")[1]

	if path == "/" {
		_, err = conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else if strings.HasPrefix(path, "/echo/") {
		res.body, _ = strings.CutPrefix(path, "/echo/")
		res.status = "OK"
		res.statusCode = 200
		res.contentType = "text/plain"
		_, err = conn.Write(res.make())
	} else if strings.HasPrefix(path, "/user-agent") {
		_, agent, _ := strings.Cut(string(reqBuff), "User-Agent:")
		agent = strings.Trim(agent, "\r\n ")
		res.body = strings.Split(agent, "\r\n")[0]
		res.status = "OK"
		res.statusCode = 200
		_, err = conn.Write(res.make())
	} else if strings.HasPrefix(path, "/files") {
		if *req.dir == "" {
			log.Println("server runs without file support")
			_, _ = conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			conn.Close()
			return
		}
		if _, err = os.Stat(*req.dir); os.IsNotExist(err) {
			log.Println("this endpoint need directory flag to be given")
			_, _ = conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			conn.Close()
			return
		}
		_, filename, _ := strings.Cut(path, "/files/")
		dat, err := os.ReadFile(*req.dir + filename)
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
		conn.Close()
	} else {
		_, err = conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
	if err != nil {
		log.Fatalln("Error while writing response.")
	}
	conn.Close()
}

var port = 4221

func main() {
	dir := flag.String("directory", "", "directory contains files")
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
