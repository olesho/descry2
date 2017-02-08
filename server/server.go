// xpatterns project xpatterns.go
package server

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/olesho/descry"
	"github.com/olesho/descry/parser"

	"github.com/gorilla/mux"
)

var patterns *parser.Patterns
var log *descry.Logger

type ResponseMessage struct {
	Message string
	Details string
}

func (r *ResponseMessage) ToJSON() []byte {
	data, _ := json.Marshal(r)
	return data
}

func LoadPatterns() error {
	patterns = parser.NewPatterns(log)
	return patterns.LoadTree(patterns.HtmlPatternTree, "patterns")
}

func Start(port int) error {
	/*
		file, _ := os.OpenFile("error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		defer file.Close()
		logger = log.New(file, "", log.Ldate|log.Ltime|log.Lshortfile)
		logger.Println("Startig server on localhost:", port)
	*/
	log := descry.NewLogger()
	log.Level = descry.LEVEL_DEBUG

	err := LoadPatterns()
	if err != nil {
		log.Message(err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/pattern/{path:.+}", handlePattern).Methods("GET", "PUT", "DELETE")
	router.HandleFunc("/patterns", listPattern)
	router.HandleFunc("/reload", reloadPatterns)
	router.HandleFunc("/parse", parseData)
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets/"))))
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
	if req.Method == "POST" {
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
}
