// +build go1.7

package xlog_test

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/justinas/alice"
	my_xlog "github.com/kanmu/xlog"
	"github.com/rs/xlog"
)

func Example_handler() {
	c := alice.New()

	host, _ := os.Hostname()
	conf := my_xlog.Config{
		// Set some global env fields
		Fields: xlog.F{
			"role": "my-service",
			"host": host,
		},
	}

	// Install the logger handler with default output on the console
	c = c.Append(my_xlog.NewHandler(conf))

	// Plug the xlog handler's input to Go's default logger
	log.SetFlags(0)
	log.SetOutput(my_xlog.New(conf))

	// Install some provided extra handler to set some request's context fields.
	// Thanks to those handler, all our logs will come with some pre-populated fields.
	c = c.Append(my_xlog.RemoteAddrHandler("ip"))
	c = c.Append(my_xlog.UserAgentHandler("user_agent"))
	c = c.Append(my_xlog.RefererHandler("referer"))
	c = c.Append(my_xlog.RequestIDHandler("req_id", "Request-Id"))

	// Here is your final handler
	h := c.Then(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the logger from the request's context. You can safely assume it
		// will be always there: if the handler is removed, xlog.FromContext
		// will return a NopLogger
		l := my_xlog.FromRequest(r)

		// Then log some errors
		if err := errors.New("some error from elsewhere"); err != nil {
			l.Errorf("Here is an error: %v", err)
		}

		// Or some info with fields
		l.Info("Something happend", xlog.F{
			"user":   "current user id",
			"status": "ok",
		})
	}))
	http.Handle("/", h)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.SetOutput(os.Stderr) // make sure we print to console
		log.Fatal(err)
	}
}
