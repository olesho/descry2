// xpatterns project xpatterns.go
package server

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/olesho/descry"
	"github.com/olesho/descry/parser"
	"gopkg.in/xmlpath.v2"

	"github.com/gorilla/mux"
)

var patterns *parser.Patterns
var log *descry.Logger
var samples []*xmlpath.Node

type ResponseMessage struct {
	Message string
	Details string
	Payload interface{}
}

func (r *ResponseMessage) ToJSON() []byte {
	data, _ := json.Marshal(r)
	return data
}

func LoadPatterns() error {
	patterns = parser.NewPatterns(log)
	return patterns.LoadTree(patterns.HtmlPatternTree, "patterns")
}

func LoadSamples() error {
	samples = []*xmlpath.Node{}
	files, err := ioutil.ReadDir("samples")
	if err != nil {
		return err
	}
	for _, f := range files {
		if !f.IsDir() {
			file, err := os.Open("samples/" + f.Name())
			if err != nil {
				return err
			}
			node, err := xmlpath.ParseHTML(file)
			if err != nil {
				return err
			}
			samples = append(samples, node)
		}
	}
	return nil
}

func Start(port int) error {
	/*
		file, _ := os.OpenFile("error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		defer file.Close()
		logger = log.New(file, "", log.Ldate|log.Ltime|log.Lshortfile)
		logger.Println("Startig server on localhost:", port)
	*/
	log = descry.NewLogger()
	log.Level = descry.LEVEL_DEBUG

	err := LoadPatterns()
	if err != nil {
		log.Message(err)
	}

	err = LoadSamples()
	if err != nil {
		log.Message(err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/pattern/{path:.+}", handlePattern).Methods("GET", "PUT", "DELETE")
	router.HandleFunc("/patterns", listPattern)
	router.HandleFunc("/reload", reloadPatterns)
	router.HandleFunc("/parse", parseData).Methods("POST")
	router.HandleFunc("/parse-gzip", parseGzip).Methods("POST")
	router.HandleFunc("/check", checkPattern).Methods("POST")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./assets/")))
	return http.ListenAndServe(":"+strconv.Itoa(port), router)
}

func reloadPatterns(res http.ResponseWriter, req *http.Request) {
	err := LoadPatterns()
	if err != nil {
		log.Message(err)
		res.WriteHeader(500)
		return
	}
	res.WriteHeader(200)
}

func parseData(res http.ResponseWriter, req *http.Request) {
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		res.Write([]byte("Unable to read request body"))
		return
	}

	if req.Header.Get("X-Source") == "" {
		log.Message("Warning: X-Source http header can't be empty")
	}

	node, err := patterns.Apply(req.Header.Get("X-Source"), bytes.NewReader(data))
	if err != nil {
		res.Write([]byte("Error applying patterns"))
		return
	}

	res.Header().Set("Content-Type", "application/json")
	json.NewEncoder(res).Encode(&node)
}

func parseGzip(res http.ResponseWriter, req *http.Request) {
	gzipReader, err := gzip.NewReader(req.Body)
	if err != nil {
		res.Write([]byte("Unable to read request body"))
		return
	}

	data, err := ioutil.ReadAll(gzipReader)
	if err != nil {
		res.Write([]byte("Unable to read request body"))
		return
	}

	if req.Header.Get("X-Source") == "" {
		log.Message("Warning: X-Source http header can't be empty")
	}

	node, err := patterns.Apply(req.Header.Get("X-Source"), bytes.NewReader(data))
	if err != nil {
		res.Write([]byte("Error applying patterns"))
		return
	}

	res.Header().Set("Content-Type", "application/json")
	json.NewEncoder(res).Encode(&node)
}

func checkPattern(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		returnMsg(res, "Unable to read HTTP request for pattern", err.Error())
		return
	}

	next_pattern := &parser.HtmlMap{}
	err = next_pattern.Unmarshal(data)
	if err != nil {
		returnMsg(res, "Unable to parse XML pattern", err.Error())
		return
	}

	// set default type for root element
	if next_pattern.Field.Type == nil {
		next_pattern.Field.Type = &parser.Type{Name: "struct"}
	}

	err = next_pattern.Compile()
	if err != nil {
		returnMsg(res, "Pattern compilation error: ", err.Error())
		return
	}

	result := []interface{}{}
	for _, sample := range samples {
		result = append(result, next_pattern.ApplyHtml("", sample))
	}
	if len(result) > 0 {
		m := &ResponseMessage{Payload: result}
		res.Write(m.ToJSON())
		return
	}
	returnMsg(res, "Empty result", "")
	return
}
