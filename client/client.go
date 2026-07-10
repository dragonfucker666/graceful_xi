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

func dialUtlsHttp2(netAddr string) (roundTripperCloserType, error) {
	tcpConn, err := net.Dial("tcp", netAddr)
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
			pr, pw := io.Pipe()
			req, err := http.NewRequest(http.MethodPost, remoteUrl, pr)
			if err != nil {
				log.Println(err)
				return
			}
			remoteWriter := pw
			resp, err := roundTripper.RoundTrip(req)
			if err != nil {
				log.Println(err)
				return
			}
			remoteReader := resp.Body
			wg := sync.WaitGroup{}
			wg.Go(func() {
				trasher.Dirty(localConn, remoteWriter)
			})
			wg.Go(func() {
				trasher.Clean(remoteReader, localConn)
			})
			wg.Wait()
		}()
	}
}

func main() {
	gxiListen := getenv.GetEnvOrDefault("GXI_LISTEN", "127.0.0.1:1080")
	gxiSendPointer := getenv.GetEnvOrPanic("GXI_SEND")
	netAddr, httpPath := parseGxiPointer(gxiSendPointer)
	listener, err := net.Listen("tcp", gxiListen)
	if err != nil {
		log.Panicln(err)
	}
	roundTripperCloser, err := dialUtlsHttp2(netAddr)
	if err != nil {
		log.Panicln(err)
	}
	defer roundTripperCloser.Close()
	log.Println("Listening on " + gxiListen)
	listen(netAddr, httpPath, listener, roundTripperCloser)
}
