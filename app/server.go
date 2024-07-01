package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer l.Close()
	log.Printf("Starting HTTP server on port: %d\n", 4221)

	req := make([]byte, 1024)

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		_, err = conn.Read(req)
		if err != nil {
			log.Fatalf("%s", err)
		}

		path := strings.Split(string(req), "\r\n")[0]
		path = strings.Split(path, " ")[1]

		if path == "/" {
			_, err = conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		} else {
			_, err = conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		}
		if err != nil {
			log.Println("Error while writing response.")
			os.Exit(1)
		}
		conn.Close()
	}
}
