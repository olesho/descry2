// server
package main

import (
	"bufio"
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/elazarl/goproxy"
	//	"github.com/gorilla/mux"
)

type ProxyInterceptor struct {
	handler func(header, body *bytes.Buffer)
}

func NewProxyInterceptor(h func(header, body *bytes.Buffer)) *ProxyInterceptor {
	return &ProxyInterceptor{h}
}

func orPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func (i *ProxyInterceptor) Listen() {
	proxy := goproxy.NewProxyHttpServer()
	proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*$"))).
		HandleConnect(goproxy.AlwaysMitm)
		// enable curl -p for all hosts on port 80

	proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*:80$"))).
		HijackConnect(func(req *http.Request, client net.Conn, ctx *goproxy.ProxyCtx) {
			defer func() {
				if e := recover(); e != nil {
					ctx.Logf("error connecting to remote: %v", e)
					client.Write([]byte("HTTP/1.1 500 Cannot reach destination\r\n\r\n"))
				}
				client.Close()
			}()
			clientBuf := bufio.NewReadWriter(bufio.NewReader(client), bufio.NewWriter(client))
			remote, err := net.Dial("tcp", req.URL.Host)
			orPanic(err)
			remoteBuf := bufio.NewReadWriter(bufio.NewReader(remote), bufio.NewWriter(remote))
			for {
				req, err := http.ReadRequest(clientBuf.Reader)
				orPanic(err)
				orPanic(req.Write(remoteBuf))
				orPanic(remoteBuf.Flush())
				resp, err := http.ReadResponse(remoteBuf.Reader, req)
				orPanic(err)
				orPanic(resp.Write(clientBuf.Writer))
				orPanic(clientBuf.Flush())
			}
		})

	proxy.OnResponse().DoFunc(func(r *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if ctx != nil {
			if ctx.Resp != nil {
				if strings.Contains(ctx.Resp.Header.Get("Content-Type"), "text/html") {

					buf, _ := ioutil.ReadAll(ctx.Resp.Body)
					rdr1 := ioutil.NopCloser(bytes.NewBuffer(buf))
					rdr2 := ioutil.NopCloser(bytes.NewBuffer(buf))
					ctx.Resp.Body = rdr2

					var header, body []byte
					headerBuffer := bytes.NewBuffer(header)
					bodyBuffer := bytes.NewBuffer(body)
					ctx.Req.WriteProxy(headerBuffer)
					io.Copy(bodyBuffer, rdr1)
					rdr1.Close()

					i.handler(headerBuffer, bodyBuffer)
				}
			}
		}

		return r
	})

	verbose := flag.Bool("v", false, "should every proxy request be logged to stdout")
	addr := flag.String("addr", ":8080", "proxy listen address")
	flag.Parse()
	proxy.Verbose = *verbose

	//go func() {
	log.Fatal(http.ListenAndServe(*addr, proxy))
	//}()

	/*
		controller := mux.NewRouter()
		controller.HandleFunc("/test", func(res http.ResponseWriter, req *http.Request) {
			res.Write([]byte("Hello test"))
			res.WriteHeader(200)
		})
		log.Fatal(http.ListenAndServeTLS(":	8081", "cert/publickey.cer", "cert/private.key", controller))
	*/
}
