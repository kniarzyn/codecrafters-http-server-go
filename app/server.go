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

func handleRoot(res *HTTPResponse, req HTTPRequest) {
	res.statusCode = 200
	res.status = "OK"
}

func handleFiles(res *HTTPResponse, req HTTPRequest) {
	if *dir == "" {
		log.Println("server runs without file support")
		res.status = "Not Found"
		res.statusCode = 404
		return
	}
	if _, err := os.Stat(*dir); os.IsNotExist(err) {
		log.Println("this endpoint need directory flag to be given")
		res.status = "Not Found"
		res.statusCode = 404
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
			res.status = "Not Found"
			res.statusCode = 404
			return
		}
		res.body = dat
		res.statusCode = 200
		res.status = "OK"
		res.contentType = "application/octet-stream"
		res.contentLength = len(res.body)
	}
}

func handleEcho(res *HTTPResponse, req HTTPRequest) {
	body, _ := strings.CutPrefix(req.path, "/echo/")
	if strings.Contains(req.headers["accept-encoding"], "gzip") {
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
}

func handleUserAgent(res *HTTPResponse, req HTTPRequest) {
	res.body = []byte(req.headers["user-agent"])
	res.contentLength = len(res.body)
	res.status = "OK"
	res.statusCode = 200
	res.contentType = "text/plain"
}

func handleConnection(conn net.Conn) {
	reqBuff := make([]byte, 32)
	rawReq := []byte{}
	res := HTTPResponse{}
	defer conn.Close()
	for {
		n, err := conn.Read(reqBuff)
		rawReq = append(rawReq, reqBuff[:n]...)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%s", err)
		}
	}
	req := parseRequest(string(rawReq))
	log.Printf("Req: %+v\n", req)

	switch {
	case req.path == "/":
		handleRoot(&res, req)
	case strings.HasPrefix(req.path, "/echo/"):
		handleEcho(&res, req)
	case strings.HasPrefix(req.path, "/user-agent"):
		handleUserAgent(&res, req)
	case strings.HasPrefix(req.path, "/files") && *dir != "":
		handleFiles(&res, req)
	default:
		res.status = "Not Found"
		res.statusCode = 404
	}
	io.Copy(conn, res)
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
		go handleConnection(conn)
	}
}
