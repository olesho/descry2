// test project main.go
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/olesho/descry2/parser"
)

func main() {
	verbose := flag.Bool("v", false, "Should every proxy request be logged to stdout")
	port := flag.String("p", os.Getenv("PORT"), "Proxy listen port address")
	patternsDir := flag.String("d", os.Getenv("PATTERNS_DIR"), "Patterns directory")
	logFileName := flag.String("l", os.Getenv("LOG"), "Log")
	flag.Parse()

	// default if no env nor flag set
	if len(*port) == 0 {
		*port = "5000"
	}

	var logger *log.Logger
	if len(*logFileName) > 0 {
		logFile, err := os.OpenFile(*logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		logger = log.New(logFile, "", log.Lshortfile)
	} else {
		logger = log.New(os.Stdout, "", log.Lshortfile)
	}

	// default if no env nor flag set
	if len(*patternsDir) == 0 {
		*patternsDir = "patterns"
	}
	patterns := parser.NewPatterns(logger)
	err := patterns.LoadTree(patterns.Tree, *patternsDir)
	if err != nil {
		logger.Panic(err)
	}

	interceptor := NewProxyInterceptor(func(header, body *bytes.Buffer) io.ReadCloser {
		// proxy handler
		url := string(regexp.MustCompile(`(GET|POST|PUT|HEAD|DELETE|OPTIONS)\s+(.+)\s+(HTTP)`).FindAllSubmatch(header.Bytes(), -1)[0][2])
		node, err := patterns.Apply(url, body)
		if err != nil {
			logger.Println("Error applying patterns: ", err.Error())
			return nil
		}

		if node == nil {
			node = make(map[string]interface{})
		}
		recognized, err := json.Marshal(&node)
		if err != nil {
			logger.Println("Error marshalling to JSON: ", err.Error())
		}

		return ioutil.NopCloser(bytes.NewBuffer(recognized))

	}, func(w http.ResponseWriter, r *http.Request) {
		// controller
		// used to realod patterns
		patterns = parser.NewPatterns(logger)
		err := patterns.LoadTree(patterns.Tree, *patternsDir)
		if err != nil {
			logger.Panic(err)
		}
	})
	log.Panic(interceptor.Listen(*port, *verbose))
}
