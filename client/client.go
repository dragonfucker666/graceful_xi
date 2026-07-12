package main

import (
	"graceful_xi/getenv"
	"graceful_xi/trasher"
	"io"
	"net"
	"net/http"
	"strings"
	"log"
	"sync"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

type roundTripperCloserType interface {
	http.RoundTripper
	io.Closer
}

func dialUtlsHttp2(netAddr string, gxiSendPort string) (roundTripperCloserType, error) {
	tcpConn, err := net.Dial("tcp", netAddr + ":" + gxiSendPort)
	if err != nil {
		return nil, err
	}
	utlsConfig := utls.Config{
		ServerName: netAddr,
	}
	tlsConn := utls.UClient(tcpConn, &utlsConfig, utls.HelloFirefox_Auto)
	http2Transport := http2.Transport{}
	return http2Transport.NewClientConn(tlsConn)
}

func parseGxiPointer(gxiPointer string) (netAddr string, httpPath string) {
	netAddr, httpPath, ok := strings.Cut(gxiPointer, "/")
	_ = ok
	return netAddr, "/" + httpPath
}

func listen(netAddr string, httpPath string, listener net.Listener, roundTripper http.RoundTripper) {
	remoteUrl := "https://" + netAddr + httpPath
	for {
		localConn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go func(){
			defer localConn.Close()
			pr, remoteWriter := io.Pipe()
			req, err := http.NewRequest(http.MethodPost, remoteUrl, pr)
			if err != nil {
				log.Println(err)
				return
			}
			resp, err := roundTripper.RoundTrip(req)
			if err != nil {
				log.Println(err)
				return
			}
			remoteReader := resp.Body
			wg := sync.WaitGroup{}
			wg.Go(func() {
				trasher.Dirty(localConn, remoteWriter)
				remoteWriter.Close()
			})
			wg.Go(func() {
				trasher.Clean(remoteReader, localConn)
				localConn.Close()
			})
			wg.Wait()
		}()
	}
}

func main() {
	gxiListen := getenv.GetEnvOrDefault("GXI_LISTEN", "127.0.0.1:1080")
	gxiSendPort := getenv.GetEnvOrDefault("GXI_SEND_PORT", "443")
	gxiSendPointer := getenv.GetEnvOrPanic("GXI_SEND")
	netAddr, httpPath := parseGxiPointer(gxiSendPointer)
	listener, err := net.Listen("tcp", gxiListen)
	if err != nil {
		log.Panicln(err)
	}
	defer listener.Close()
	roundTripperCloser, err := dialUtlsHttp2(netAddr, gxiSendPort)
	if err != nil {
		log.Panicln(err)
	}
	defer roundTripperCloser.Close()
	log.Println("Listening on " + gxiListen)
	listen(netAddr, httpPath, listener, roundTripperCloser)
}
