package xlog

import (
	"io"
	"io/ioutil"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/rs/xlog"
	"github.com/stretchr/testify/assert"
)

var fakeNow = time.Date(0, 0, 0, 0, 0, 0, 0, time.Local)
var critialLoggerMux = sync.Mutex{}

func init() {
}

func TestNew(t *testing.T) {
	oc := NewOutputChannel(newTestOutput())
	defer oc.Close()
	c := Config{
		Level:     xlog.LevelError,
		Output:    oc,
		Fields:    xlog.F{"foo": "bar"},
		NowGetter: func() time.Time { return fakeNow },
	}
	L := New(c)
	l, ok := L.(*logger)
	if assert.True(t, ok) {
		assert.Equal(t, xlog.LevelError, l.level)
		assert.Equal(t, c.Output, l.output)
		assert.Equal(t, xlog.F{"foo": "bar"}, xlog.F(l.fields))
		// Ensure l.fields is a clone
		c.Fields["bar"] = "baz"
		assert.Equal(t, xlog.F{"foo": "bar"}, xlog.F(l.fields))
		assert.Equal(t, false, l.disablePooling)
		l.close()
	}
}

func TestNewPoolDisabled(t *testing.T) {
	oc := NewOutputChannel(newTestOutput())
	defer oc.Close()
	originalPool := loggerPool
	defer func(p *sync.Pool) {
		loggerPool = originalPool
	}(originalPool)
	loggerPool = &sync.Pool{
		New: func() interface{} {
			assert.Fail(t, "pool used when disabled")
			return nil
		},
	}
	c := Config{
		Level:          xlog.LevelError,
		Output:         oc,
		Fields:         xlog.F{"foo": "bar"},
		DisablePooling: true,
		NowGetter:      func() time.Time { return fakeNow },
	}
	L := New(c)
	l, ok := L.(*logger)
	if assert.True(t, ok) {
		assert.Equal(t, xlog.LevelError, l.level)
		assert.Equal(t, c.Output, l.output)
		assert.Equal(t, xlog.F{"foo": "bar"}, xlog.F(l.fields))
		// Ensure l.fields is a clone
		c.Fields["bar"] = "baz"
		assert.Equal(t, xlog.F{"foo": "bar"}, xlog.F(l.fields))
		assert.Equal(t, true, l.disablePooling)
		l.close()
		// Assert again to ensure close does not remove internal state
		assert.Equal(t, xlog.LevelError, l.level)
		assert.Equal(t, c.Output, l.output)
		assert.Equal(t, xlog.F{"foo": "bar"}, xlog.F(l.fields))
		// Ensure l.fields is a clone
		c.Fields["bar"] = "baz"
		assert.Equal(t, xlog.F{"foo": "bar"}, xlog.F(l.fields))
		assert.Equal(t, true, l.disablePooling)
	}
}

func TestCopy(t *testing.T) {
	oc := NewOutputChannel(newTestOutput())
	defer oc.Close()
	c := Config{
		Level:     xlog.LevelError,
		Output:    oc,
		Fields:    xlog.F{"foo": "bar"},
		NowGetter: func() time.Time { return fakeNow },
	}
	l := New(c).(*logger)
	l2 := Copy(l).(*logger)
	assert.Equal(t, l.output, l2.output)
	assert.Equal(t, l.level, l2.level)
	assert.Equal(t, l.fields, l2.fields)
	l2.SetField("bar", "baz")
	assert.Equal(t, xlog.F{"foo": "bar"}, l.fields)
	assert.Equal(t, xlog.F{"foo": "bar", "bar": "baz"}, l2.fields)

	assert.Equal(t, NopLogger, Copy(NopLogger))
	assert.Equal(t, NopLogger, Copy(nil))
}

func TestNewDefautOutput(t *testing.T) {
	L := New(Config{NowGetter: func() time.Time { return fakeNow }})
	l, ok := L.(*logger)
	if assert.True(t, ok) {
		assert.NotNil(t, l.output)
		l.close()
	}
}

func TestSend(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o, NowGetter: func() time.Time { return fakeNow }}).(*logger)
	l.send(xlog.LevelDebug, 1, "test", xlog.F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "debug", "message": "test", "foo": "bar"}, last)

	l.SetField("bar", "baz")
	l.send(xlog.LevelInfo, 1, "test", xlog.F{"foo": "bar"})
	last = <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "info", "message": "test", "foo": "bar", "bar": "baz"}, last)

	l = New(Config{Output: o, Level: 1}).(*logger)
	o.reset()
	l.send(0, 2, "test", xlog.F{"foo": "bar"})
	assert.True(t, o.empty())
}

func TestSendDrop(t *testing.T) {
	t.Skip()
	r, w := io.Pipe()
	go func() {
		critialLoggerMux.Lock()
		defer critialLoggerMux.Unlock()
		oldCritialLogger := critialLogger
		critialLogger = log.New(w, "", 0)
		o := newTestOutput()
		oc := NewOutputChannelBuffer(Discard, 1)
		l := New(Config{Output: oc, NowGetter: func() time.Time { return fakeNow }}).(*logger)
		l.send(xlog.LevelDebug, 2, "test", xlog.F{"foo": "bar"})
		l.send(xlog.LevelDebug, 2, "test", xlog.F{"foo": "bar"})
		l.send(xlog.LevelDebug, 2, "test", xlog.F{"foo": "bar"})
		o.get()
		o.get()
		o.get()
		oc.Close()
		critialLogger = oldCritialLogger
		w.Close()
	}()
	b, err := ioutil.ReadAll(r)
	assert.NoError(t, err)
	assert.Contains(t, string(b), "send error: buffer full")
}

func TestExtractFields(t *testing.T) {
	v := []interface{}{"a", 1, map[string]interface{}{"foo": "bar"}}
	f := extractFields(&v)
	assert.Equal(t, map[string]interface{}{"foo": "bar"}, f)
	assert.Equal(t, []interface{}{"a", 1}, v)

	v = []interface{}{map[string]interface{}{"foo": "bar"}, "a", 1}
	f = extractFields(&v)
	assert.Nil(t, f)
	assert.Equal(t, []interface{}{map[string]interface{}{"foo": "bar"}, "a", 1}, v)

	v = []interface{}{"a", 1, xlog.F{"foo": "bar"}}
	f = extractFields(&v)
	assert.Equal(t, map[string]interface{}{"foo": "bar"}, f)
	assert.Equal(t, []interface{}{"a", 1}, v)

	v = []interface{}{}
	f = extractFields(&v)
	assert.Nil(t, f)
	assert.Equal(t, []interface{}{}, v)
}

func TestGetFields(t *testing.T) {
	oc := NewOutputChannelBuffer(Discard, 1)
	l := New(Config{Output: oc, NowGetter: func() time.Time { return fakeNow }}).(*logger)
	l.SetField("k", "v")
	assert.Equal(t, xlog.F{"k": "v"}, l.GetFields())
}

func TestDebug(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o, NowGetter: func() time.Time { return fakeNow }}).(*logger)
	l.Debug("test", xlog.F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "debug", "message": "test", "foo": "bar"}, last)
}

func TestDebugf(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o, NowGetter: func() time.Time { return fakeNow }}).(*logger)
	l.Debugf("test %d", 1, xlog.F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "debug", "message": "test 1", "foo": "bar"}, last)
}

func TestInfo(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o, NowGetter: func() time.Time { return fakeNow }}).(*logger)
	l.Info("test", xlog.F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "info", "message": "test", "foo": "bar"}, last)
}

func TestInfof(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o, NowGetter: func() time.Time { return fakeNow }}).(*logger)
	l.Infof("test %d", 1, xlog.F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "info", "message": "test 1", "foo": "bar"}, last)
}

func TestWarn(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o, NowGetter: func() time.Time { return fakeNow }}).(*logger)
	l.Warn("test", xlog.F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "warn", "message": "test", "foo": "bar"}, last)
}

func TestWarnf(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o, NowGetter: func() time.Time { return fakeNow }}).(*logger)
	l.Warnf("test %d", 1, xlog.F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "warn", "message": "test 1", "foo": "bar"}, last)
}

func TestError(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o, NowGetter: func() time.Time { return fakeNow }}).(*logger)
	l.Error("test", xlog.F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "error", "message": "test", "foo": "bar"}, last)
}

func TestErrorf(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o, NowGetter: func() time.Time { return fakeNow }}).(*logger)
	l.Errorf("test %d%v", 1, xlog.F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "error", "message": "test 1", "foo": "bar"}, last)
}

func TestFatal(t *testing.T) {
	e := exit1
	exited := 0
	exit1 = func() { exited++ }
	defer func() { exit1 = e }()
	o := newTestOutput()
	l := New(Config{Output: NewOutputChannel(o), NowGetter: func() time.Time { return fakeNow }}).(*logger)
	l.Fatal("test", xlog.F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "fatal", "message": "test", "foo": "bar"}, last)
	assert.Equal(t, 1, exited)
}

func TestFatalf(t *testing.T) {
	e := exit1
	exited := 0
	exit1 = func() { exited++ }
	defer func() { exit1 = e }()
	o := newTestOutput()
	l := New(Config{Output: NewOutputChannel(o), NowGetter: func() time.Time { return fakeNow }}).(*logger)
	l.Fatalf("test %d%v", 1, xlog.F{"foo": "bar"})
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "fatal", "message": "test 1", "foo": "bar"}, last)
	assert.Equal(t, 1, exited)
}

func TestWrite(t *testing.T) {
	o := newTestOutput()
	xl := New(Config{Output: NewOutputChannel(o), NowGetter: func() time.Time { return fakeNow }}).(*logger)
	l := log.New(xl, "prefix ", 0)
	l.Printf("test")
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "info", "message": "prefix test"}, last)
}

func TestOutput(t *testing.T) {
	o := newTestOutput()
	l := New(Config{Output: o, NowGetter: func() time.Time { return fakeNow }}).(*logger)
	l.Output(2, "test")
	last := <-o.w
	assert.Contains(t, last["file"], "log_test.go:")
	delete(last, "file")
	assert.Equal(t, map[string]interface{}{"time": fakeNow, "level": "info", "message": "test"}, last)
}
