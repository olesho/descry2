// descry-app project main.go
package main

import (
	"fmt"

	"github.com/olesho/descry/server"
)

func main() {
	port := 9999
	fmt.Println("Descry recognition server started at :", port)
	err := server.Start(port)
	if err != nil {
		panic(err)
	}
}
