package main

import (
	"log"
	"net"
	"sync"
	"graceful_xi/getenv"
	"graceful_xi/trasher"
)

func main() {
	gxiListen := getenv.GetEnvOrPanic("GXI_LISTEN")
	gxiSend := getenv.GetEnvOrPanic("GXI_SEND")
	listener, err := net.Listen("tcp", gxiListen)
	if err != nil {
		log.Panicln(err)
	}
	defer listener.Close()
	for {
		listenConn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go func(){
			defer listenConn.Close()
			sendConn, err := net.Dial("tcp", gxiSend)
			if err != nil {
				log.Println(err)
				return
			}
			defer sendConn.Close()
			wg := sync.WaitGroup{}
			wg.Go(func() {
				trasher.Clean(listenConn, sendConn)
			})
			wg.Go(func() {
				trasher.Dirty(sendConn, listenConn)
			})
			wg.Wait()
		}()
	}
}
