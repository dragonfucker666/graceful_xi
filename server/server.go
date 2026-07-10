package main

import (
	"log"
	"net"
	"graceful_xi/getenv"
)

func main() {
	gxiListen := getenv.GetEnvOrPanic("GXI_LISTEN")
	gxiSend := getenv.GetEnvOrPanic("GXI_SEND")
	listener, err := net.Listen("tcp", gxiListen)
	if err != nil {
		log.Panicln(err)
	}
	for {
		listenConn, err := listener.Accept()
	}
}
