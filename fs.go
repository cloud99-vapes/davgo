package davgo

import (
	"bytes"
	"fmt"
	"github.com/moovweb/gokogiri"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"time"
)

type Session struct {
	base     *url.URL
	client   http.Client
	username string
	password string
}

type FileInfo struct {
	Href  string
	Size  int
	IsDir bool
	Stamp time.Time
}

type PropFindRes struct {
	Fi []FileInfo
}

func (p *PropFindRes) Parse(b []byte) (err error) {
	xml, err := gokogiri.ParseXml(b)
	root := xml.Root()
	root.RecursivelyRemoveNamespaces()
	res, _ := root.Search("response")
	for _, i := range res {
		var finfo FileInfo
		href, _ := i.Search("href")
		if href == nil {
			continue
		}
		finfo.Href = href[0].Content()
		sz, _ := i.Search("propstat/prop/getcontentlength")
		if sz == nil {
			continue
		}
		finfo.Size, _ = strconv.Atoi(sz[0].Content())
		col, _ := i.Search("propstat/prop/resourcetype/collection")
		if col != nil {
			finfo.IsDir = true
		} else {
			finfo.IsDir = false
		}
		mt, _ := i.Search("propstat/prop/creationdate")
		if mt != nil {
			finfo.Stamp, _ = time.Parse(time.RFC3339, mt[0].Content())
		}
		p.Fi = append(p.Fi, finfo)
	}
	return
}

func (p *PropFindRes) ToRelative(base *url.URL) {
	for i, m := range p.Fi {
		u, _ := filepath.Rel(base.String(), m.Href)
		p.Fi[i].Href = u
	}
	return
}

func NewSession(rooturl string) (s *Session, err error) {
	jar, _ := cookiejar.New(nil)
	cl := http.Client{Jar: jar}
	u, _ := url.Parse(rooturl)
	s = &Session{u, cl, "", ""}
	return
}

func (s *Session) SetBasicAuth(user, pass string) {
	s.username = user
	s.password = pass
	return
}

func (s *Session) Chdir(name string) (err error) {
	nexturl, err := s.base.Parse(name)
	if nexturl != nil {
		s.base = nexturl
	}
	return
}

func (s *Session) Abs(name string) (res string) {
	u, _ := url.Parse(s.base.String())
	u.Path = path.Join(u.Path, name)
	leng := len(name)
	if name[leng-1:leng] == "/" {
		return u.String() + "/"
	}
	return u.String()
}

func (s *Session) NewRequest(method, name string) (req *http.Request, err error) {
	req, err = http.NewRequest(method, s.Abs(name), nil)
	if err != nil {
		return
	}
	req.Host = s.base.Host
	if s.username != "" {
		req.SetBasicAuth(s.username, s.password)
	}
	return
}

func (s *Session) DoRequest(req *http.Request) (res *http.Response, err error) {
	res, err = s.client.Do(req)
	return
}

func (s *Session) Res2Err(res *http.Response, success []int) (err error) {
	for _, v := range success {
		if v == res.StatusCode {
			return nil
		}
	}
	return fmt.Errorf("%d %s", res.StatusCode, res.Status)
}

func (s *Session) Listdir(name string) (fi []FileInfo, err error) {
	req, err := s.NewRequest("PROPFIND", name)
	req.Header.Add("depth", "1")
	req.Header.Add("translate", "f")
	res, err := s.DoRequest(req)
	resbody, err := ioutil.ReadAll(res.Body)
	err = s.Res2Err(res, []int{200})
	if err != nil {
		p := PropFindRes{}
		p.Parse(resbody)
		p.ToRelative(s.base)
		fi = p.Fi
	}
	return
}

func (s *Session) Stat(name string) (fi FileInfo, err error) {
	req, err := s.NewRequest("PROPFIND", name)
	req.Header.Add("depth", "0")
	req.Header.Add("translate", "f")
	res, err := s.DoRequest(req)
	resbody, err := ioutil.ReadAll(res.Body)
	err = s.Res2Err(res, []int{200})
	if err != nil {
		p := PropFindRes{}
		p.Parse(resbody)
		fi = p.Fi[0]
	}
	return
}

func (s *Session) Rename(name, dest string) (err error) {
	req, err := s.NewRequest("MOVE", name)
	req.Header.Add("Destination", s.Abs(dest))
	res, err := s.DoRequest(req)
	_, err = ioutil.ReadAll(res.Body)
	err = s.Res2Err(res, []int{201})
	return
}

func (s *Session) remove(name string, depth string) (err error) {
	req, err := s.NewRequest("DELETE", name)
	req.Header.Add("Depth", depth)
	res, err := s.DoRequest(req)
	_, err = ioutil.ReadAll(res.Body)
	err = s.Res2Err(res, []int{204})
	return
}

func (s *Session) Copy(name, dest string) (err error) {
	req, err := s.NewRequest("COPY", name)
	req.Header.Add("Destination", s.Abs(dest))
	res, err := s.DoRequest(req)
	_, err = ioutil.ReadAll(res.Body)
	err = s.Res2Err(res, []int{201, 204})
	return
}

func (s *Session) Mkdir(name string) (err error) {
	req, err := s.NewRequest("MKCOL", name)
	res, err := s.DoRequest(req)
	_, err = ioutil.ReadAll(res.Body)
	err = s.Res2Err(res, []int{201})
	return
}

func (s *Session) Remove(name string) (err error) {
	return s.remove(name, "0")
}

func (s *Session) Rmdir(name string) (err error) {
	return s.remove(name, "1")
}

func (s *Session) RmR(name string) (err error) {
	return s.remove(name, "infinity")
}

func (s *Session) Lock(name string) (token string, err error) {
	return
}

func (s *Session) UnLock(name, token string) (err error) {
	return
}

func (s *Session) NewReader(name string) (rd *io.ReadCloser, err error) {
	req, err := s.NewRequest("GET", name)
	res, err := s.DoRequest(req)
	return &res.Body, err
}

func (s *Session) Put(name string, data []byte) (err error) {
	req, err := http.NewRequest("PUT", s.Abs(name), bytes.NewBuffer(data))
	if err != nil {
		return
	}
	req.Host = s.base.Host
	if s.username != "" {
		req.SetBasicAuth(s.username, s.password)
	}
	req.ContentLength = int64(len(data))
	res, err := s.DoRequest(req)
	err = s.Res2Err(res, []int{201, 204})
	return
}

func (s *Session) PutRange(name string, off int64, data []byte) (err error) {
	req, err := http.NewRequest("PUT", s.Abs(name), bytes.NewBuffer(data))
	dlen := int64(len(data))
	rg := fmt.Sprintf("bytes %d-%d/%d", off, dlen+off, dlen+off+1)
	req.Header.Add("Content-Range", rg)
	req.Host = s.base.Host
	if s.username != "" {
		req.SetBasicAuth(s.username, s.password)
	}
	req.ContentLength = dlen
	res, err := s.DoRequest(req)
	err = s.Res2Err(res, []int{201, 204})
	return
}
