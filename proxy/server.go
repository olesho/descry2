// server
package main

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/elazarl/goproxy"
)

type ProxyInterceptor struct {
	proxyHandler   func(header, body *bytes.Buffer) io.ReadCloser
	controlHandler func(w http.ResponseWriter, r *http.Request)
}

func NewProxyInterceptor(h func(header, body *bytes.Buffer) io.ReadCloser, c func(w http.ResponseWriter, r *http.Request)) *ProxyInterceptor {
	return &ProxyInterceptor{h, c}
}

func orPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func (i *ProxyInterceptor) Listen(port string, verbose bool) error {
	proxy := goproxy.NewProxyHttpServer()

	proxy.NonproxyHandler = http.HandlerFunc(http.HandlerFunc(i.controlHandler))

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
					//rdr2 := ioutil.NopCloser(bytes.NewBuffer(buf))
					//ctx.Resp.Body = rdr2

					var header, body []byte
					headerBuffer := bytes.NewBuffer(header)
					bodyBuffer := bytes.NewBuffer(body)
					ctx.Req.WriteProxy(headerBuffer)
					io.Copy(bodyBuffer, rdr1)
					rdr1.Close()

					ctx.Resp.Body = i.proxyHandler(headerBuffer, bodyBuffer)
				}
			}
		}

		return r
	})

	proxy.Verbose = verbose
	return http.ListenAndServe(":"+port, proxy)
}
