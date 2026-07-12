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

type restartingHttp2ClientConnType struct {
	roundTripperCloserType
	mutex sync.Mutex
	netAddr string
	gxiSendPort string
}

type roundTripperCloserType interface {
	http.RoundTripper
	io.Closer
}

func (c *restartingHttp2ClientConnType) RoundTrip(req *http.Request) (*http.Response, error) {
	firstTime := true
	for {
		resp, err := c.roundTripperCloserType.RoundTrip(req)
		if err != nil && firstTime {
			if err.Error() == "http2: client conn not usable" || err.Error() == "http2: client conn could not be established" {
				c.Close()
				c.mutex.Lock()
				c.roundTripperCloserType, err = dialUtlsHttp2(c.netAddr, c.gxiSendPort)
				if err != nil {
					return nil, err
				}
				c.mutex.Unlock()
				firstTime = false
				continue
			}
		}
		return resp, err
	}
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
