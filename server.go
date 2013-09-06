// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strings"
)

// ----------------------------------------------------------------------------
// Codec
// ----------------------------------------------------------------------------

// Codec creates a CodecRequest to process each request.
type Codec interface {
	NewRequest(*http.Request) CodecRequest
}

// CodecRequest decodes a request and encodes a response using a specific
// serialization scheme.
type CodecRequest interface {
	// Reads request and returns the RPC method name.
	Method() (string, error)
	// Reads request filling the RPC method args.
	ReadRequest(interface{}) error
	// Writes response using the RPC method reply. The error parameter is
	// the error returned by the method call, if any.
	WriteResponse(http.ResponseWriter, interface{}, error) error
}

// ----------------------------------------------------------------------------
// Server
// ----------------------------------------------------------------------------

var (
	ErrEmptyBindLocal    = errors.New("rpc: local address list is empty")
	ErrMalformedRemoteIp = errors.New("rpc: remote client rejected, cannot read its IP")
	ErrRemoteNotAllowed  = errors.New("rpc: remote client rejected, not allowed by the server")
)

// NewServer returns a new RPC server.
func NewServer() *Server {
	return &Server{
		codecs:   make(map[string]Codec),
		services: new(serviceMap),
	}
}

// Server serves registered RPC services using registered codecs.
type Server struct {
	codecs   map[string]Codec
	services *serviceMap
	allow    []net.IP
}

// RegisterCodec adds a new codec to the server.
//
// Codecs are defined to process a given serialization scheme, e.g., JSON or
// XML. A codec is chosen based on the "Content-Type" header from the request,
// excluding the charset definition.
func (s *Server) RegisterCodec(codec Codec, contentType string) {
	s.codecs[strings.ToLower(contentType)] = codec
}

// RegisterService adds a new service to the server.
//
// The name parameter is optional: if empty it will be inferred from
// the receiver type name.
//
// Methods from the receiver will be extracted if these rules are satisfied:
//
//    - The receiver is exported (begins with an upper case letter) or local
//      (defined in the package registering the service).
//    - The method name is exported.
//    - The method has three arguments: *http.Request, *args, *reply.
//    - All three arguments are pointers.
//    - The second and third arguments are exported or local.
//    - The method has return type error.
//
// All other methods are ignored.
func (s *Server) RegisterService(receiver interface{}, name string) error {
	return s.services.register(receiver, name)
}

// HasMethod returns true if the given method is registered.
//
// The method uses a dotted notation as in "Service.Method".
func (s *Server) HasMethod(method string) bool {
	if _, _, err := s.services.get(method); err == nil {
		return true
	}
	return false
}

// Bind makes the server to only accept requests comming from
// specified IP addresses.
func (s *Server) Bind(allow ...net.IP) {
	s.allow = allow
}

// BindLocal makes the server to accept requests comming from
// local IP only.
func (s *Server) BindLocal() (err error) {
	var addrs []net.Addr
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return
	}
	local := make([]net.IP, 0, len(addrs))
	for i := range addrs {
		if ip, ok := addrs[i].(*net.IPNet); ok {
			local = append(local, ip.IP)
		}
	}
	if len(local) == 0 {
		return ErrEmptyBindLocal
	}
	s.Bind(local...)
	return
}

// ServeHTTP
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := s.clientAllowed(r.RemoteAddr); err != nil {
		writeError(w, 403, err.Error())
		return
	}
	if r.Method != "POST" {
		writeError(w, 405, "rpc: POST method required, received "+r.Method)
		return
	}
	contentType := r.Header.Get("Content-Type")
	idx := strings.Index(contentType, ";")
	if idx != -1 {
		contentType = contentType[:idx]
	}
	codec := s.codecs[strings.ToLower(contentType)]
	if codec == nil {
		writeError(w, 415, "rpc: unrecognized Content-Type: "+contentType)
		return
	}
	// Create a new codec request.
	codecReq := codec.NewRequest(r)
	// Get service method to be called.
	method, errMethod := codecReq.Method()
	if errMethod != nil {
		writeError(w, 400, errMethod.Error())
		return
	}
	serviceSpec, methodSpec, errGet := s.services.get(method)
	if errGet != nil {
		writeError(w, 400, errGet.Error())
		return
	}
	// Decode the args.
	args := reflect.New(methodSpec.argsType)
	if errRead := codecReq.ReadRequest(args.Interface()); errRead != nil {
		writeError(w, 400, errRead.Error())
		return
	}
	// Call the service method.
	reply := reflect.New(methodSpec.replyType)
	errValue := methodSpec.method.Func.Call([]reflect.Value{
		serviceSpec.rcvr,
		reflect.ValueOf(r),
		args,
		reply,
	})
	// Cast the result to error if needed.
	var errResult error
	errInter := errValue[0].Interface()
	if errInter != nil {
		errResult = errInter.(error)
	}
	// Prevents Internet Explorer from MIME-sniffing a response away
	// from the declared content-type
	w.Header().Set("x-content-type-options", "nosniff")
	// Encode the response.
	if errWrite := codecReq.WriteResponse(w, reply.Interface(), errResult); errWrite != nil {
		writeError(w, 400, errWrite.Error())
	}
}

func (s *Server) clientAllowed(remoteAddr string) (err error) {
	if len(s.allow) == 0 {
		return nil
	}
	var (
		host string
		ip   net.IP
	)
	if host, _, err = net.SplitHostPort(remoteAddr); err != nil {
		return fmt.Errorf("%s: %s", ErrMalformedRemoteIp, err)
	}
	if ip = net.ParseIP(host); ip == nil {
		return ErrMalformedRemoteIp
	}
	for i := range s.allow {
		if s.allow[i].Equal(ip) {
			return nil
		}
	}
	return ErrRemoteNotAllowed
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, msg)
}
