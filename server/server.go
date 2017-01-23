// xpatterns project xpatterns.go
package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"bitbucket.org/olesho/scrub/recognizer/parser"

	"github.com/gorilla/mux"
)

var patterns *parser.Patterns

type ResponseMessage struct {
	Message string
	Details string
}

func (r *ResponseMessage) ToJSON() []byte {
	data, _ := json.Marshal(r)
	return data
}

func LoadPatterns() error {
	patterns = parser.NewPatterns()
	return patterns.LoadTree(patterns.HtmlPatternTree, "patterns")
}

func Start(port int) error {
	err := LoadPatterns()
	if err != nil {
		fmt.Println(err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/pattern/{title}", handlePattern)
	router.HandleFunc("/patterns", listPattern)
	router.HandleFunc("/reload", reloadPatterns)
	router.HandleFunc("/parse", parseData)
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets/"))))
	return http.ListenAndServe(":"+strconv.Itoa(port), router)
}

func reloadPatterns(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Reloading patterns...")
	err := LoadPatterns()
	if err != nil {
		fmt.Println(err)
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
			fmt.Println("Warning: X-Source http header can't be empty")
		} else {
			fmt.Println(req.Header.Get("X-Source"))
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
