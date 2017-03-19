// test project main.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/olesho/descry2/parser"
)

func main() {
	logger := log.New(os.Stdout, "", log.Lshortfile)
	patterns := parser.NewPatterns(logger)
	err := patterns.LoadTree(patterns.Tree, "patterns")
	if err != nil {
		logger.Panic(err)
	}

	var getRequestUrl = regexp.MustCompile(`(GET|POST|PUT|HEAD|DELETE|OPTIONS)\s+(.+)\s+(HTTP)`)
	interceptor := NewProxyInterceptor(func(header, body *bytes.Buffer) {
		url := string(getRequestUrl.FindAllSubmatch(header.Bytes(), -1)[0][2])
		node, err := patterns.Apply(url, body)
		if err != nil {
			logger.Println("Error applying patterns: ", err.Error())
			return
		}

		if node != nil {
			recognized, err := json.Marshal(&node)
			if err != nil {
				logger.Println("Error marshalling to JSON: ", err.Error())
			}

			fmt.Println(string(recognized))
		}
	})
	interceptor.Listen()
}
