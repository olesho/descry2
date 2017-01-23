// descry project pattern.go
package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
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
	vars := mux.Vars(req)
	title := vars["title"]
	switch req.Method {
	case "GET":
		if title != "" {
			data, err := ReadPattern(title)
			if err != nil {
				returnMsg(res, "Unable to find pattern "+title, err.Error())
				return
			}
			res.Header().Set("Content-Type", "text/xml")
			res.Write(data)
		}
	case "PUT":
		if title != "" {
			data, err := ioutil.ReadAll(req.Body)
			if err != nil {
				returnMsg(res, "Unable to read HTTP request for pattern "+title, err.Error())
				return
			}

			err = WritePattern(title, data)
			if err != nil {
				returnMsg(res, "Unable to write pattern "+title, err.Error())
				return
			}

			returnMsg(res, "Pattern "+title+" written succesfully", "")
			return
		}
		returnMsg(res, "Please provide correct Title for a pattern", "")

	case "DELETE":
		RemovePattern(title)
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
