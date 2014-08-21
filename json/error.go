// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2012 The Gorilla Authors. All rights reserved.
// Copyright 2013 X-Formation. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"encoding/json"
	"fmt"
)

// Error allows for passing an JSON object to an error field
// in a server's response.
type Error struct {
	object map[string]interface{}
	blob   json.RawMessage
}

// NewErrorBlob creates a wrapper for a JSON interface value. It can be used by either
// a service's handler func to write more complex JSON data to an error field
// of a server's response, or by a client to read it.
func NewErrorBlob(blob []byte) (e *Error, err error) {
	e = &Error{
		object: make(map[string]interface{}),
		blob:   blob,
	}
	if err = json.Unmarshal(e.blob, &e.object); err != nil {
		return nil, err
	}
	return
}

// NewErrorObject creates a wrapper for a JSON interface value. It can be used by either
// a service's handler func to write more complex JSON data to an error field
// of a server's response, or by a client to read it.
func NewErrorObject(object map[string]interface{}) (e *Error) {
	var err error
	e = &Error{
		object: object,
	}
	if e.blob, err = json.Marshal(e.Object()); err != nil {
		e.blob = nil
		e.object = map[string]interface{}{
			"code":    -1,
			"message": fmt.Sprintf("%+v", object),
		}
	}
	return
}

// Error
func (e Error) Error() string {
	return string([]byte(e.blob))
}

// Object
func (e Error) Object() map[string]interface{} {
	return e.object
}
