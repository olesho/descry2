// descry-app project main.go
package main

import (
	"flag"

	"github.com/olesho/descry/server"
)

func main() {
	var port = flag.Int("p", 5000, "Listen port")
	flag.Parse()
	err := server.Start(*port)
	if err != nil {
		panic(err)
	}
}
