// +build go1.7

package xlog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/rs/xlog"
	"github.com/stretchr/testify/assert"
)

func TestFromContext(t *testing.T) {
	assert.Equal(t, NopLogger, FromContext(nil))
	assert.Equal(t, NopLogger, FromContext(context.Background()))
	l := &logger{}
	ctx := NewContext(context.Background(), l)
	assert.Equal(t, l, FromContext(ctx))
}

func TestNewHandler(t *testing.T) {
	c := Config{
		Level:  xlog.LevelInfo,
		Fields: xlog.F{"foo": "bar"},
		Output: NewOutputChannel(&testOutput{}),
	}
	lh := NewHandler(c)
	h := lh(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r)
		assert.NotNil(t, l)
		assert.NotEqual(t, NopLogger, l)
		if l, ok := l.(*logger); assert.True(t, ok) {
			assert.Equal(t, xlog.LevelInfo, l.level)
			assert.Equal(t, c.Output, l.output)
			assert.Equal(t, xlog.F{"foo": "bar"}, xlog.F(l.fields))
		}
	}))
	h.ServeHTTP(nil, &http.Request{})
}

func TestURLHandler(t *testing.T) {
	r := &http.Request{
		URL: &url.URL{Path: "/path", RawQuery: "foo=bar"},
	}
	h := URLHandler("url")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r).(*logger)
		assert.Equal(t, xlog.F{"url": "/path?foo=bar"}, xlog.F(l.fields))
	}))
	h = NewHandler(Config{})(h)
	h.ServeHTTP(nil, r)
}

func TestMethodHandler(t *testing.T) {
	r := &http.Request{
		Method: "POST",
	}
	h := MethodHandler("method")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r).(*logger)
		assert.Equal(t, xlog.F{"method": "POST"}, xlog.F(l.fields))
	}))
	h = NewHandler(Config{})(h)
	h.ServeHTTP(nil, r)
}

func TestRequestHandler(t *testing.T) {
	r := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/path", RawQuery: "foo=bar"},
	}
	h := RequestHandler("request")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r).(*logger)
		assert.Equal(t, xlog.F{"request": "POST /path?foo=bar"}, xlog.F(l.fields))
	}))
	h = NewHandler(Config{})(h)
	h.ServeHTTP(nil, r)
}

func TestRemoteAddrHandler(t *testing.T) {
	r := &http.Request{
		RemoteAddr: "1.2.3.4:1234",
	}
	h := RemoteAddrHandler("ip")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r).(*logger)
		assert.Equal(t, xlog.F{"ip": "1.2.3.4"}, xlog.F(l.fields))
	}))
	h = NewHandler(Config{})(h)
	h.ServeHTTP(nil, r)
}

func TestRemoteAddrHandlerIPv6(t *testing.T) {
	r := &http.Request{
		RemoteAddr: "[2001:db8:a0b:12f0::1]:1234",
	}
	h := RemoteAddrHandler("ip")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r).(*logger)
		assert.Equal(t, xlog.F{"ip": "2001:db8:a0b:12f0::1"}, xlog.F(l.fields))
	}))
	h = NewHandler(Config{})(h)
	h.ServeHTTP(nil, r)
}

func TestUserAgentHandler(t *testing.T) {
	r := &http.Request{
		Header: http.Header{
			"User-Agent": []string{"some user agent string"},
		},
	}
	h := UserAgentHandler("ua")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r).(*logger)
		assert.Equal(t, xlog.F{"ua": "some user agent string"}, xlog.F(l.fields))
	}))
	h = NewHandler(Config{})(h)
	h.ServeHTTP(nil, r)
}

func TestRefererHandler(t *testing.T) {
	r := &http.Request{
		Header: http.Header{
			"Referer": []string{"http://foo.com/bar"},
		},
	}
	h := RefererHandler("ua")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r).(*logger)
		assert.Equal(t, xlog.F{"ua": "http://foo.com/bar"}, xlog.F(l.fields))
	}))
	h = NewHandler(Config{})(h)
	h.ServeHTTP(nil, r)
}

func TestRequestIDHandler(t *testing.T) {
	r := &http.Request{}
	h := RequestIDHandler("id", "Request-Id")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r).(*logger)
		if id, ok := IDFromRequest(r); assert.True(t, ok) {
			assert.Equal(t, l.fields["id"], id)
			assert.Len(t, id.String(), 20)
			assert.Equal(t, id.String(), w.Header().Get("Request-Id"))
		}
		assert.Len(t, l.fields["id"], 12)
	}))
	h = NewHandler(Config{})(h)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
}
