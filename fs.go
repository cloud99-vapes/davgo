package davgo

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"io"
	"io/ioutil"
	"log"
	"encoding/xml"
)

type Session struct {
	base   *url.URL
	client http.Client
}

type FileInfo struct {
	Href string `xml:"href"`
	Size int    `xml:"propstat>prop>getcontentlength"`
	IsDir string  `xml:"propstat>prop>resourcetype>collection,directive"`
}

type PropFindRes struct {
	Fi []FileInfo `xml:"response"`
}

func NewSession(rooturl string) (s *Session, err error) {
	jar, _ := cookiejar.New(nil)
	cl := http.Client{Jar: jar}
	u, _ := url.Parse(rooturl)
	s = &Session{u, cl}
	return
}

func (s *Session) NewRequest(method, name string) (req *http.Request, err error) {
	u,_ := url.Parse(s.base.String())
	u.Path = path.Join(u.Path, name)
	log.Println("url", u.String(), s.base.String(), name)
	req, err = http.NewRequest(method, u.String(), nil)
	if err != nil {
		return
	}
	req.Host=s.base.Host
	return
}

func (s *Session) DoRequest(req *http.Request) (res *http.Response, err error) {
	res, err = s.client.Do(req)
	return
}

func (s *Session) Listdir(name string) (fi []FileInfo, err error) {
	req, err := s.NewRequest("PROPFIND", name)
	req.Header.Add("depth", "1")
	req.Header.Add("translate", "f")
	res, err := s.DoRequest(req)
	resbody, err := ioutil.ReadAll(res.Body)
	var v PropFindRes
	err = xml.Unmarshal(resbody, &v)
	log.Println("listdir v1", v.Fi, string(resbody))
	fi=v.Fi
	return
}

func (s *Session) Stat(name string) (fi FileInfo, err error) {
	return
}

func (s *Session) Rename(name, dest string) (err error) {
	return
}

func (s *Session) Remove(name string) (err error) {
	return
}

func (s *Session) Copy(name, dest string) (err error) {
	return
}

func (s *Session) Mkdir(name string) (err error) {
	return
}

func (s *Session) Rmdir(name string) (err error) {
	return
}

func (s *Session) NewReader(name string) (rd *io.ReadCloser, err error) {
	req, err := s.NewRequest("GET", name)
	res, err := s.DoRequest(req)
	return &res.Body, err
}

func (s *Session) NewWriter(name string) (wr *io.Writer, err error) {
	return
}
