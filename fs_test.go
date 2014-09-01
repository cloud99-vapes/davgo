package davgo

import (
	"io/ioutil"
	"testing"
)

func TestFS(t *testing.T) {
	s, err := NewSession("http://localhost:8008/go")
	t.Log("Session", s, err)
	r, err := s.NewRequest("GET", "src/test")
	t.Log("Request", r, err)
	res, err := s.DoRequest(r)
	t.Log("Response", res, err)
	rd, err := s.NewReader("src/test")
	t.Log("Reader", rd, err)
	body, err := ioutil.ReadAll(*rd)
	t.Log("Body", string(body), err)
	fi, err := s.Listdir("src/test")
	t.Log("Listdir", fi, err)
	err = s.Copy("testfile.txt", "testfile2.txt")
	t.Log("Copy", err)
	err = s.Remove("testfile2.txt")
	t.Log("Remove", err)
}
