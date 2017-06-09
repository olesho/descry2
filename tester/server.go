// server
package main

import (
	"bufio"
	"bytes"
	"flag"
	//	"fmt"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/elazarl/goproxy"
	"github.com/gorilla/mux"
	"github.com/olesho/descry2/parser"
	"gopkg.in/xmlpath.v2"
)

var getRequestUrl = regexp.MustCompile(`(GET|POST|PUT|HEAD|DELETE|OPTIONS)\s+(.+)\s+(HTTP)`)

type TestServer struct {
	handler func(header, body *bytes.Buffer)
	storage *BoltStorage
}

func NewTestServer() (*TestServer, error) {
	storage, err := NewBoltStorage()
	if err != nil {
		return nil, err
	}

	h := func(header, body *bytes.Buffer) {
		url := string(getRequestUrl.FindAllSubmatch(header.Bytes(), -1)[0][2])
		storage.SaveBody(url, body.Bytes())
	}

	return &TestServer{h, storage}, nil
}

func orPanic(err error) {
	if err != nil {
		panic(err)
	}
}

// list current urls
func (i *TestServer) ListJson() []byte {
	r := make([]byte, 0)
	r = append(r, []byte("{\"list\": [\n")...)
	list := make([]string, 0)
	i.storage.ListBody(func(k string, v []byte) {
		list = append(list, "\""+k+"\"")
	})
	r = append(r, []byte(strings.Join(list, ",\n")+"\n]}")...)
	return r
}

func (i *TestServer) Listen() {
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
	ui_addr := flag.String("ui_addr", ":8081", "UI listen address")
	flag.Parse()
	proxy.Verbose = *verbose

	go func() {
		log.Fatal(http.ListenAndServe(*addr, proxy))
	}()

	ui := mux.NewRouter()

	ui.HandleFunc("/list", func(res http.ResponseWriter, req *http.Request) {
		res.Write(i.ListJson())
		res.Header().Set("Content-Type", "application/json")
	})

	ui.HandleFunc("/list/flush", func(res http.ResponseWriter, req *http.Request) {
		err := i.storage.Flush()
		if err != nil {
			res.Write(response(nil, err))
		}
		res.Write(i.ListJson())
		res.Header().Set("Content-Type", "application/json")
	})

	ui.HandleFunc("/list/add", func(res http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		byteUrl, err := ioutil.ReadAll(req.Body)
		url := string(byteUrl)
		if err != nil {
			res.Write(response("", err))
			return
		}

		client := &http.Client{}
		r, err := http.NewRequest("GET", url, nil)
		if err != nil {
			res.Write(response("", err))
			return
		}
		r.Header.Set("Upgrade-Insecure-Request", "1")
		r.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/54.0.2840.71 Safari/537.36")
		r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Header.Set("Accept-Encoding", "gzip, deflate, sdch, br")
		r.Header.Set("Accept-Language", "uk-UA,uk;q=0.8,ru;q=0.6,en-US;q=0.4,en;q=0.2")
		rr, err := client.Do(r)
		if err != nil {
			res.Write(response("", err))
			return
		}
		defer rr.Body.Close()
		i.storage.SaveResponse(url, rr)

		//res.Write(response(nil, err))
		res.Write(i.ListJson())
		res.Header().Set("Content-Type", "application/json")
	})

	ui.HandleFunc("/check", func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "application/json")

		data, err := ioutil.ReadAll(req.Body)
		if err != nil {
			res.Write(response("", err))
			return
		}

		next_pattern := &parser.Map{}
		err = next_pattern.Unmarshal(data)
		if err != nil {
			res.Write(response("", err))
			return
		}

		nextCompiled, err := next_pattern.Compile()
		if err != nil {
			res.Write(response("", err))
			return
		}

		result := []interface{}{}
		i.storage.ListBody(func(k string, v []byte) {
			buf := bytes.NewBuffer(v)
			node, err := xmlpath.ParseHTML(buf)
			if err != nil {
				log.Println(err)
			}
			r := nextCompiled.ApplyHtml(k, node)
			if r != nil {
				result = append(result, r)
			}
		})

		if len(result) > 0 {
			res.Write(response(result, nil))
			return
		}
		res.Write(response("", nil))
		return
	})

	ui.PathPrefix("/").Handler(http.FileServer(http.Dir("./assets/")))

	log.Fatal(http.ListenAndServe(*ui_addr, ui))
}

type SuccessStruct struct {
	Data interface{}
}

type FailStruct struct {
	Error string
}

func response(data interface{}, err error) []byte {
	var r []byte
	if err != nil {
		r, _ = json.Marshal(&FailStruct{err.Error()})
	} else {
		r, _ = json.Marshal(&SuccessStruct{data})
	}
	return r
}
