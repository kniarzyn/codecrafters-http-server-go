package main

import (
	"fmt"
	"log"
	"net"
	"strings"
)

var port = 4221

func main() {
	address := fmt.Sprintf(":%d", port)
	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to bind to port %d\n", port)
	}

	defer l.Close()
	log.Printf("Starting HTTP server on port: %d\n", port)

	req := make([]byte, 1024)
	for {
		conn, err := l.Accept()
		if err != nil {
      log.Fatalln("Error accepting connection: ", err.Error())
		}
		_, err = conn.Read(req)
		if err != nil {
			log.Fatalf("%s", err)
		}

		path := strings.Split(string(req), " ")[1]

		if path == "/" {
			_, err = conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		} else if strings.HasPrefix(path, "/echo/") {
			echoText, _ := strings.CutPrefix(path, "/echo/")
			status := "OK"
			statusCode := 200
			response := fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", statusCode, status, len(echoText), echoText)
			_, err = conn.Write([]byte(response))
		} else {
			_, err = conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		}
		if err != nil {
			log.Fatalln("Error while writing response.")
		}
		conn.Close()
	}
}
