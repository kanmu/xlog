package xlog_test

import (
	my_xlog "github.com/kanmu/xlog"
	"github.com/rs/xlog"
)

func ExampleSetLogger() {
	my_xlog.SetLogger(my_xlog.New(xlog.Config{
		Level:  xlog.LevelInfo,
		Output: my_xlog.NewConsoleOutput(),
		Fields: xlog.F{
			"role": "my-service",
		},
	}))
}
