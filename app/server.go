package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

type HTTPRequest struct {
	conn    net.Conn
	dir     *string
	headers map[string]string
	method  string
	path    string
	body    string
}

type HTTPResponse struct {
	protocol        string
	body            []byte
	status          string
	contentType     string
	contentEncoding string
	contentLength   int
	statusCode      int
	readBytes       int
}

func (r HTTPResponse) Read(buf []byte) (int, error) {
	if r.protocol == "" {
		r.protocol = "HTTP/1.1"
	}
	headers := fmt.Sprintf(
		"Content-Type: %s\r\nContent-Length: %d",
		r.contentType,
		r.contentLength,
	)
	if r.contentEncoding != "" {
		headers = fmt.Sprintf("%s\r\nContent-Encoding: %s", headers, r.contentEncoding)
	}
	s := []byte(fmt.Sprintf("%s %d %s\r\n%s\r\n\r\n%s", r.protocol, r.statusCode, r.status, headers, r.body))

	if len(s) >= cap(buf) {
		copy(buf, s[r.readBytes:cap(buf)])
		return cap(buf), nil
	} else {
		copy(buf, s[r.readBytes:])
		return len(s), io.EOF
	}
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

	return req
}

func commpressBody(body string) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write([]byte(body))
	if err != nil {
		return nil, err
	}
	zw.Close()
	return buf.Bytes(), nil
}

func handleFiles(res HTTPResponse, req HTTPRequest) {
	if *dir == "" {
		log.Println("server runs without file support")
		_, _ = req.conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		return
	}
	if _, err := os.Stat(*dir); os.IsNotExist(err) {
		log.Println("this endpoint need directory flag to be given")
		_, _ = req.conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		return
	}

	_, filename, _ := strings.Cut(req.path, "/files/")
	filePath := *dir + "/" + filename
	if req.method == "POST" {
		log.Printf("parsing POST request with file path: %s", filePath)
		fmt.Printf("Body: %v", req.headers["content-length"])
		bodyLength, _ := strconv.Atoi(req.headers["content-length"])
		dat := []byte(req.body)[:bodyLength]
		err := os.WriteFile(filePath, dat, 0666)
		if err != nil {
			log.Printf("error creating file: %s \n%s", filePath, err)
		}
		res.body = dat
		res.statusCode = 201
		res.status = "Created"
		return
	} else {
		dat, err := os.ReadFile(filePath)
		if err != nil {
			_, _ = req.conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			return
		}
		res.body = dat
		res.statusCode = 200
		res.status = "OK"
		res.contentType = "application/octet-stream"
		res.contentLength = len(res.body)
	}
}

func handleConnection(req HTTPRequest) {
	reqBuff := make([]byte, 1024)
	res := HTTPResponse{}
	conn := req.conn
	defer conn.Close()
	_, err := conn.Read(reqBuff)
	if err != nil {
		log.Fatalf("%s", err)
	}
	r := parseRequest(string(reqBuff))

	switch {
	case r.path == "/":
		io.Copy(conn, strings.NewReader("HTTP/1.1 200 OK\r\n\r\n"))
	case strings.HasPrefix(r.path, "/echo/"):
		body, _ := strings.CutPrefix(r.path, "/echo/")
		if strings.Contains(r.headers["accept-encoding"], "gzip") {
			res.contentEncoding = "gzip"
			cbody, _ := commpressBody(body)
			res.body = cbody
		} else {
			res.body = []byte(body)
		}
		res.status = "OK"
		res.statusCode = 200
		res.contentType = "text/plain"
		res.contentLength = len(res.body)
		io.Copy(conn, res)
	case strings.HasPrefix(r.path, "/user-agent"):
		res.body = []byte(r.headers["user-agent"])
		res.contentLength = len(res.body)
		res.status = "OK"
		res.statusCode = 200
		res.contentType = "text/plain"
		io.Copy(conn, res)
	case strings.HasPrefix(r.path, "/files") && *dir != "":
		handleFiles(res, r)
	default:
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
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
