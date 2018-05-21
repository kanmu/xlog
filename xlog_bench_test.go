package xlog

import (
	"github.com/rs/xlog"
	"testing"
)

func BenchmarkSend(b *testing.B) {
	l := New(xlog.Config{Output: Discard, Fields: xlog.F{"a": "b"}}).(*logger)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		l.send(0, 0, "test", xlog.F{"foo": "bar", "bar": "baz"})
	}
}
