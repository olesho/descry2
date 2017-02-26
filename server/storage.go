// xpatterns project xpatterns.go
package server

import (
	//	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func RemovePattern(title string) error {
	path := "patterns/" + title

	fi, err := os.Stat(path)
	if err != nil {
		return err
	}

	if fi.Mode().IsDir() {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}

func WritePattern(title string, data []byte) error {
	path := "patterns/" + title
	dir := filepath.Dir(path)
	if is, err := exists(dir); !is || err != nil {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return ioutil.WriteFile("patterns/"+title, data, 0755)
}

func ReadPattern(title string) ([]byte, error) {
	data, err := ioutil.ReadFile("patterns/" + title)
	return data, err
}
