// descry project pattern.go
package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/olesho/descry/parser"
)

func listPattern(res http.ResponseWriter, req *http.Request) {
	list, err := ListPatterns()
	if err != nil {
		res.Write([]byte("Unable to read pattern list"))
	}
	data := strings.Join(list, "\n")
	res.Write([]byte(data))
}

func handlePattern(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("Cache-Control", "no-cache, no-store, must-revalidate")
	res.Header().Add("Expires", "0")

	vars := mux.Vars(req)
	path := vars["path"]
	switch req.Method {
	case "GET":
		if path != "" {
			data, err := ReadPattern(path)
			if err != nil {
				returnMsg(res, "Unable to find pattern "+path, err.Error())
				return
			}
			res.Header().Set("Content-Type", "text/xml")
			res.Write(data)
		}
	case "PUT":
		if path != "" {
			data, err := ioutil.ReadAll(req.Body)
			if err != nil {
				returnMsg(res, "Unable to read HTTP request for pattern "+path, err.Error())
				return
			}

			next_pattern := &parser.HtmlMap{}
			next_pattern.Unmarshal(data)

			// set default type for root element
			if next_pattern.Field.Type == nil {
				next_pattern.Field.Type = &parser.Type{Name: "struct"}
			}

			err = next_pattern.Compile()
			if err != nil {
				returnMsg(res, "Pattern compilation error: ", err.Error())
				return
			}

			err = WritePattern(path, data)
			if err != nil {
				returnMsg(res, "Unable to write pattern "+path, err.Error())
				return
			}

			returnMsg(res, "Pattern "+path+" written succesfully", "")
			return
		}
		returnMsg(res, "Please provide correct Title for a pattern", "")

	case "DELETE":
		RemovePattern(path)
	}
}

func returnMsg(res http.ResponseWriter, msg string, details string) {
	fmt.Println(msg)
	m := &ResponseMessage{
		Message: msg,
		Details: details,
	}
	res.Write(m.ToJSON())
	return
}
