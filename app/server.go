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

func handleConnection(req HTTPRequest) {
	reqBuff := make([]byte, 1024)
	conn := req.conn
	_, err := conn.Read(reqBuff)
	if err != nil {
		log.Fatalf("%s", err)
	}

	path := strings.Split(string(reqBuff), " ")[1]

	if path == "/" {
		_, err = conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else if strings.HasPrefix(path, "/echo/") {
		echoText, _ := strings.CutPrefix(path, "/echo/")
		status := "OK"
		statusCode := 200
		response := fmt.Sprintf(
			"HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
			statusCode,
			status,
			len(echoText),
			echoText,
		)
		_, err = conn.Write([]byte(response))
	} else if strings.HasPrefix(path, "/user-agent") {
		fmt.Printf("%s", string(reqBuff))
		_, agent, _ := strings.Cut(string(reqBuff), "User-Agent:")
		agent = strings.Trim(agent, "\r\n ")
		agent = strings.Split(agent, "\r\n")[0]
		status := "OK"
		statusCode := 200
		response := fmt.Sprintf(
			"HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
			statusCode,
			status,
			len(agent),
			agent,
		)
		_, err = conn.Write([]byte(response))
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
		res := fmt.Sprintf(
			"HTTP/1.1 %d %s\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s",
			200,
			"OK",
			len(dat),
			string(dat),
		)
		_, _ = conn.Write([]byte(res))
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
