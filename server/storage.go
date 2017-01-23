// xpatterns project xpatterns.go
package server

import (
	"io/ioutil"
	"os"
)

func RemovePattern(title string) error {
	return os.Remove("patterns/" + title)
}

func WritePattern(title string, data []byte) error {
	return ioutil.WriteFile("patterns/"+title, data, 0755)
}

func ReadPattern(title string) ([]byte, error) {
	data, err := ioutil.ReadFile("patterns/" + title)
	return data, err
}

func ListPatterns() ([]string, error) {
	files, err := ioutil.ReadDir("patterns")
	if err != nil {
		return nil, err
	}
	result := make([]string, len(files))
	for i, f := range files {
		result[i] = f.Name()
	}
	return result, nil
}
