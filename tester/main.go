// test project main.go
package main

import (
	"log"
	//	"os"

	//	"github.com/olesho/descry2/parser"
)

func main() {
	//logger := log.New(os.Stdout, "", log.Lshortfile)
	//	patterns := parser.NewPatterns(logger)
	/*
		err := patterns.LoadTree(patterns.Tree, "patterns")
		if err != nil {
			logger.Panic(err)
		}
	*/

	server, err := NewTestServer()
	if err != nil {
		log.Panic(err)
		//logger.Panic(err)
	}
	server.Listen()
}
