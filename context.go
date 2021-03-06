// Copyright © 2009--2014 The Web.go Authors
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package web

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"code.google.com/p/go.net/websocket"
)

// Custom web.go request context. Contains information about the request and
// can be used to manipulate the response.
type Context struct {
	// The incoming request that led to this handler being invoked
	Request *http.Request
	RawBody []byte
	// Aggregated parameters from the query string and POST data.
	Params Params
	Server *Server
	// Copied from Server.User before the handler is invoked. Use this to
	// communicate global state between your handlers.
	User interface{}
	// The response writer that the handler should write to.
	Response *ResponseWriter
	// In the case of websocket: a reference to the connection object. Nil
	// otherwise.
	WebsockConn     *websocket.Conn
	oneaccesslogger OneAccessLogger
}

// Response headers not request headers. For clarity use
// Context.Response.Header() this method exists so Context satisfies the
// http.ResponseWriter interface.
func (ctx *Context) Header() http.Header {
	return ctx.Response.Header()
}

// Write raw data back to the client
func (ctx *Context) Write(data []byte) (int, error) {
	return ctx.Response.Write(data)
}

func (ctx *Context) WriteHeader(status int) {
	ctx.Response.WriteHeader(status)
}

// Best-effort serialization of response data
func (ctx *Context) writeAnything(i interface{}) error {
	switch typed := i.(type) {
	case string:
		_, err := ctx.Write([]byte(typed))
		return err
	case []byte:
		_, err := ctx.Write(typed)
		return err
	case io.WriterTo:
		_, err := typed.WriteTo(ctx)
		return err
	case io.Reader:
		_, err := io.Copy(ctx, typed)
		return err
	}
	// Can't cast to a more specific type than interface{}, try encoders
	mime := ctx.Header().Get("content-type")
	if enc, ok := encoders[mime]; ok {
		return enc(ctx).Encode(i)
	}
	return errors.New("cannot serialize data for writing to client")
}

func (ctx *Context) Abort(status int, body string) {
	ctx.ContentType("txt")
	ctx.WriteHeader(status)
	ctx.Write([]byte(body))
}

func (ctx *Context) Redirect(status int, url_ string) {
	ctx.Header().Set("Location", url_)
	ctx.Abort(status, "Redirecting to: "+url_)
}

func (ctx *Context) NotModified() {
	ctx.WriteHeader(304)
}

func (ctx *Context) NotFound(message string) {
	ctx.Abort(404, message)
}

func (ctx *Context) NotAcceptable(message string) {
	ctx.Abort(406, message)
}

func (ctx *Context) Unauthorized(message string) {
	ctx.Abort(401, message)
}

func (ctx *Context) Forbidden(message string) {
	ctx.Abort(403, message)
}

// Sets the content type by extension, as defined in the mime package.
// For example, ctx.ContentType("json") sets the content-type to "application/json"
// if the supplied extension contains a slash (/) it is set as the content-type
// verbatim without passing it to mime.  returns the content type as it was
// set, or an empty string if none was found.
func (ctx *Context) ContentType(ext string) string {
	ctype := ""
	if strings.ContainsRune(ext, '/') {
		ctype = ext
	} else {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		ctype = mime.TypeByExtension(ext)
	}
	if ctype != "" {
		ctx.Header().Set("Content-Type", ctype)
	}
	return ctype
}
