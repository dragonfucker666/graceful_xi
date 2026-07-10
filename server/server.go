package main

import (
	"net/http"
	"log"
	"net"
	"sync"
	"graceful_xi/getenv"
	"graceful_xi/trasher"
)

func main() {
	gxiListen := getenv.GetEnvOrPanic("GXI_LISTEN")
	gxiSend := getenv.GetEnvOrPanic("GXI_SEND")
	http.HandleFunc("/{anything...}", func(responseWriter http.ResponseWriter, request *http.Request) {
		sendConn, err := net.Dial("tcp", gxiSend)
		if err != nil {
			log.Println(err)
			return
		}
		defer sendConn.Close()
		wg := sync.WaitGroup{}
		wg.Go(func() {
			trasher.Clean(request.Body, sendConn)
		})
		wg.Go(func() {
			trasher.Dirty(sendConn, responseWriter)
		})
		wg.Wait()
	})
	log.Fatal(http.ListenAndServe(gxiListen, nil))
}
