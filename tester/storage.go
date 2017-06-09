// test project main.go
package main

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/boltdb/bolt"
)

type BoltStorage struct {
	db *bolt.DB
}

func NewBoltStorage() (*BoltStorage, error) {
	db, err := bolt.Open("storage.db", 0600, nil)
	if err != nil {
		return nil, err
	}
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte("body"))
		return err
	})
	return &BoltStorage{db}, nil
}

func (s *BoltStorage) Close() {
	s.Close()
}

func (s *BoltStorage) SaveBody(url string, body []byte) error { //body []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("body"))
		return b.Put([]byte(url), body)
	})
}

func (s *BoltStorage) SaveResponse(url string, resp *http.Response) error { //body []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("body"))
		var err error
		var reader io.ReadCloser
		switch resp.Header.Get("Content-Encoding") {
		case "gzip":
			reader, err = gzip.NewReader(resp.Body)
			defer reader.Close()
		default:
			reader = resp.Body
		}

		body, err := ioutil.ReadAll(reader)
		if err != nil {
			return err
		}

		return b.Put([]byte(url), body)
	})
}

func (s *BoltStorage) ListBody(iterator func(key string, val []byte)) error {
	return s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("body"))

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			iterator(string(k), v)
		}

		return nil
	})
}

func (s *BoltStorage) Flush() error {

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("body"))

		var f = func(k []byte, v []byte) error {
			return b.Delete(k)
		}

		b.ForEach(f)
		return nil
	})
}
