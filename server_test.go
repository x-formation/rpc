// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2012 The Gorilla Authors. All rights reserved.
// Copyright 2013 X-Formation. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type Service1Request struct {
	A int
	B int
}

type Service1Response struct {
	Result int
}

type Service1 struct {
}

func (t *Service1) Multiply(r *http.Request, req *Service1Request, res *Service1Response) error {
	res.Result = req.A * req.B
	return nil
}

type Service2 struct {
}

func TestRegisterService(t *testing.T) {
	var err error
	s := NewServer()
	service1 := new(Service1)
	service2 := new(Service2)

	// Inferred name.
	err = s.RegisterService(service1, "")
	if err != nil || !s.HasMethod("Service1.Multiply") {
		t.Errorf("Expected to be registered: Service1.Multiply")
	}
	// Provided name.
	err = s.RegisterService(service1, "Foo")
	if err != nil || !s.HasMethod("Foo.Multiply") {
		t.Errorf("Expected to be registered: Foo.Multiply")
	}
	// No methods.
	err = s.RegisterService(service2, "")
	if err == nil {
		t.Errorf("Expected error on service2")
	}
}

type record struct {
	addr string
	ok   bool
}

func executeTable(t *testing.T, srv *Server, table []record) {
	for _, res := range table {
		req, err := http.NewRequest("POST", "http://127.0.0.1:80", strings.NewReader("request"))
		if err != nil {
			t.Fatal("expected r to be nil, got instead:", err)
		}
		req.RemoteAddr = res.addr
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		if res.ok {
			if w.Code == 403 {
				t.Errorf("expected w.Code to be different than 403 for %s", res.addr)
			}
		} else {
			if w.Code != 403 {
				t.Errorf("expected w.Code to be 403 for %s, got instead: %d", res.addr, w.Code)
			}
		}
	}
}

func TestBind(t *testing.T) {
	srv := NewServer()
	srv.Bind(
		net.IPv4(233, 100, 100, 33),
		net.IPv4(198, 65, 22, 33),
	)
	table := []record{
		{"127.0.0.1:8082", false},
		{"198.65.43.43:7900", false},
		{"233.100.100.33:8082", true},
		{"198.65.22.33:7900", true},
		{"198.65.22.33:7900", true},
		{"123.32.33.33:8080", false},
	}
	executeTable(t, srv, table)
}

func TestBindLocal(t *testing.T) {
	srv := NewServer()
	before := []record{
		{"32.32.33.33:8080", true},
		{"127.0.0.1:8082", true},
		{"[::1]:8082", true},
	}
	after := []record{
		{"127.0.0.1:8083", true},
		{"32.32.33.33:8081", false},
		{"[::1]:8084", true},
	}
	executeTable(t, srv, before)
	if err := srv.BindLocal(); err != nil {
		t.Fatal("expected err to be nil, got instead:", err)
	}
	executeTable(t, srv, after)
}
