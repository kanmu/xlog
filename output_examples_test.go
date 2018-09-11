package xlog_test

import (
	"log/syslog"

	my_xlog "github.com/kanmu/xlog"
	"github.com/rs/xlog"
)

func Example_combinedOutputs() {
	conf := my_xlog.Config{
		Output: my_xlog.NewOutputChannel(xlog.MultiOutput{
			// Output interesting messages to console
			0: xlog.FilterOutput{
				Cond: func(fields map[string]interface{}) bool {
					val, found := fields["type"]
					return found && val == "interesting"
				},
				Output: my_xlog.NewConsoleOutput(),
			},
			// Also setup by-level loggers
			1: xlog.LevelOutput{
				// Send debug messages to console if they match type
				Debug: xlog.FilterOutput{
					Cond: func(fields map[string]interface{}) bool {
						val, found := fields["type"]
						return found && val == "interesting"
					},
					Output: my_xlog.NewConsoleOutput(),
				},
			},
			// Also send everything over syslog
			2: my_xlog.NewSyslogOutput("", "", ""),
		}),
	}

	lh := my_xlog.NewHandler(conf)
	_ = lh
}

func ExampleMultiOutput() {
	conf := my_xlog.Config{
		Output: my_xlog.NewOutputChannel(xlog.MultiOutput{
			// Output everything to console
			0: my_xlog.NewConsoleOutput(),
			// and also to local syslog
			1: my_xlog.NewSyslogOutput("", "", ""),
		}),
	}
	lh := my_xlog.NewHandler(conf)
	_ = lh
}

func ExampleFilterOutput() {
	conf := my_xlog.Config{
		Output: my_xlog.NewOutputChannel(xlog.FilterOutput{
			// Match messages containing a field type = interesting
			Cond: func(fields map[string]interface{}) bool {
				val, found := fields["type"]
				return found && val == "interesting"
			},
			// Output matching messages to the console
			Output: my_xlog.NewConsoleOutput(),
		}),
	}

	lh := my_xlog.NewHandler(conf)
	_ = lh
}

func ExampleLevelOutput() {
	conf := my_xlog.Config{
		Output: my_xlog.NewOutputChannel(xlog.LevelOutput{
			// Send debug message to console
			Debug: my_xlog.NewConsoleOutput(),
			// and error messages to syslog
			Error: my_xlog.NewSyslogOutput("", "", ""),
			// other levels are discarded
		}),
	}

	lh := my_xlog.NewHandler(conf)
	_ = lh
}

func ExampleNewSyslogWriter() {
	conf := my_xlog.Config{
		Output: my_xlog.NewOutputChannel(xlog.LevelOutput{
			Debug: my_xlog.NewLogstashOutput(my_xlog.NewSyslogWriter("", "", syslog.LOG_LOCAL0|syslog.LOG_DEBUG, "")),
			Info:  my_xlog.NewLogstashOutput(my_xlog.NewSyslogWriter("", "", syslog.LOG_LOCAL0|syslog.LOG_INFO, "")),
			Warn:  my_xlog.NewLogstashOutput(my_xlog.NewSyslogWriter("", "", syslog.LOG_LOCAL0|syslog.LOG_WARNING, "")),
			Error: my_xlog.NewLogstashOutput(my_xlog.NewSyslogWriter("", "", syslog.LOG_LOCAL0|syslog.LOG_ERR, "")),
		}),
	}

	lh := my_xlog.NewHandler(conf)
	_ = lh
}
