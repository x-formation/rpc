// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2012 The Gorilla Authors. All rights reserved.
// Copyright 2013 X-Formation. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"encoding/json"
)

// Error allows for passing an JSON object to an error field
// in a server's response.
type Error struct {
	object map[string]interface{}
	blob   json.RawMessage
}

// NewError
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

// NewErrorObject
func NewErrorObject(object map[string]interface{}) (e *Error, err error) {
	e = &Error{
		object: object,
	}
	if e.blob, err = json.Marshal(e.Object()); err != nil {
		return nil, err
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
