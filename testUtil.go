package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"gotest.tools/assert"
)

var testSettings configSettings
var testlog *os.File
var cfgFile string = "./test/config.conf"

func piTestMain(m *testing.M) {
	testSettings = initSettings(cfgFile)
	testlog, _ = setupLogging(testSettings, false)

	// run the tests
	code := m.Run()
	testlog.Close()

	os.Exit(code)
}

func logCaller(pc uintptr, file string, line int, ok bool) {
	if !ok {
		file = "?"
		line = 0
	}

	fn := runtime.FuncForPC(pc)
	var fnName string
	if fn == nil {
		fnName = "?()"
	} else {
		dotName := filepath.Ext(fn.Name())
		fnName = strings.TrimLeft(dotName, ".") + "()"
	}

	log.Printf("Starting %s (%s:%d)", fnName, filepath.Base(file), line)
}

func testRuntime() (runtimeConfig, clockwork.FakeClock, commChannels) {
	// to keep wg from complaining, add extra wg every test
	// we never wg.Wait in testing so who cares
	wg.Add(1)
	// make rt for test, log the start of the test
	logCaller(runtime.Caller(1))
	rt := initTestRuntime(testSettings)
	return rt, rt.clock.(clockwork.FakeClock), rt.comms
}

func almStateRead(t *testing.T, c chan almStateMsg) (almStateMsg, error) {
	select {
	case e := <-c:
		return e, nil
	default:
		assert.Equal(t, false, "Nothing to read from alarm channel")
	}
	return almStateMsg{}, nil
}

func almStateNoRead(t *testing.T, c chan almStateMsg) (almStateMsg, error) {
	select {
	case e := <-c:
		assert.Equal(t, e == almStateMsg{}, "Got an unexpected value on alarm channel")
	default:
	}
	return almStateMsg{}, nil
}

func ledRead(t *testing.T, c chan ledEffect) (ledEffect, error) {
	select {
	case e := <-c:
		return e, nil
	default:
		assert.Equal(t, false, "Nothing to read from led channel")
	}
	return ledEffect{}, nil
}

func ledNoRead(t *testing.T, c chan ledEffect) (ledEffect, error) {
	select {
	case e := <-c:
		assert.Equal(t, e == ledEffect{}, "Got an unexpected value from led channel")
	default:
	}
	return ledEffect{}, nil
}

func effectRead(t *testing.T, c chan displayEffect) (displayEffect, error) {
	select {
	case e := <-c:
		return e, nil
	default:
		assert.Equal(t, false, "Nothing to read from effect channel")
	}
	return displayEffect{}, nil
}

func effectReads(t *testing.T, c chan displayEffect, count int) ([]displayEffect, error) {
	de := make([]displayEffect, count)
	for i := 0; i < count; i++ {
		de[i], _ = effectRead(t, c)
	}
	return de, nil
}

func effectNoRead(t *testing.T, c chan displayEffect) (displayEffect, error) {
	select {
	case e := <-c:
		assert.Equal(t, e == displayEffect{}, "Got an unexpected value from effect channel")
	default:
	}
	return displayEffect{}, nil
}

type stepCallback func(step int)

func testBlockDuration(clock clockwork.FakeClock, step time.Duration, d time.Duration) {
	testBlockDurationCB(clock, step, d, func(int) {})
}

func testBlockDurationCB(clock clockwork.FakeClock, step time.Duration, d time.Duration, cb stepCallback) {
	start := clock.Now()
	keepClicking := true
	var count int = 0
	for keepClicking {
		count++
		clock.Advance(step)
		clock.BlockUntil(1)
		// use the callback
		if cb != nil {
			cb(count)
		}
		if clock.Now().Sub(start) > d {
			keepClicking = false
		}
	}
}

func testQuit(rt runtimeConfig) {
	//	close(rt.comms.quit)
	//	rt.clock.(clockwork.FakeClock).Advance(time.Second)
}
