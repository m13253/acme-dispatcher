/*
    ACME-dispatcher -- Dispatch ACME challenge for a multihomed server
    Copyright (C) 2017 Star Brilliant <m13253@hotmail.com>

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"github.com/gorilla/handlers"
)

type server struct {
	conf		*config
	servemux	*http.ServeMux
}

func newServer(conf *config) *server {
	s := &server {
		conf: conf,
		servemux: http.NewServeMux(),
	}
	s.servemux.HandleFunc(s.conf.Path, s.handlerFunc)
	return s
}

func (s *server) Start() error {
	return http.ListenAndServe(s.conf.Listen, handlers.CombinedLoggingHandler(os.Stdout, s.servemux))
}

func (s *server) handlerFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	if r.Header.Get(s.conf.CircularPrevention) == "yes" {
		http.Error(w, "Not Found", 404)
		return
	}
	ctx, cancel := context.WithCancel(context.TODO())
	path := r.URL.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	clientAddr, err := net.ResolveTCPAddr("tcp", r.RemoteAddr)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Error", 500)
		return
	}
	respChan := make(chan *http.Response, 1)
	errChan := make(chan *http.Response, 1)
	var wg sync.WaitGroup
	for _, upstream := range s.conf.Forward {
		wg.Add(1)
		go func(ctx context.Context, upstream, path string) {
			defer wg.Done()
			req, err := http.NewRequest(r.Method, upstream + path, nil)
			if err != nil {
				log.Println(err)
				return
			}
			req = req.WithContext(ctx)
			for k, v := range r.Header {
				if k != "Accept-Encoding" && k != "Content-Encoding" && k != "Connection" && k != "Proxy-Connection" {
					req.Header[k] = v
				}
			}
			req.Header.Set("X-Real-IP", clientAddr.IP.String())
			xff := req.Header.Get("X-Forwarded-For")
			if xff != "" {
				xff = xff + "," + clientAddr.IP.String()
			} else {
				xff = clientAddr.IP.String()
			}
			req.Header.Set("X-Forwarded-For", xff)
			req.Header.Set(s.conf.CircularPrevention, "yes")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Println(err)
				return
			}
			if resp.StatusCode < 500 && resp.StatusCode != 404 {
				select {
				case respChan <- resp:
				default:
				}
			} else {
				select {
				case errChan <- resp:
				default:
				}
			}
		}(ctx, upstream, path)
	}
	go func() {
		wg.Wait()
		select {
		case errResp := <-errChan:
			select {
			case respChan <- errResp:
			default:
			}
		default:
			close(respChan)
		}
	}()
	resp, ok := <-respChan
	if !ok {
		http.Error(w, "Bad Gateway", 502)
		return
	}
	cancel()
	respHeader := w.Header()
	for k, v := range resp.Header {
		if k != "Accept-Encoding" && k != "Content-Encoding" && k != "Connection" && k != "Proxy-Connection" {
			respHeader[k] = v
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
