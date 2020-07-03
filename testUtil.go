package main

import (
	"fmt"
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
	setupLogging(testSettings, false)

	// run the tests
	code := m.Run()
	testlog.Close()

	os.Exit(code)
}

func logCaller(msg string, depth int) {
	pc, file, line, ok := runtime.Caller(depth + 1)
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

	log.Printf("%s %s (%s:%d)", msg, fnName, filepath.Base(file), line)
}

func testRuntime() (runtimeConfig, clockwork.FakeClock, commChannels) {
	// to keep wg from complaining, add extra wg every test
	// we never wg.Wait in testing so who cares
	wg.Add(1)
	// log who is starting the test
	logCaller("Starting ", 1)
	// make rt for test, log the start of the test
	rt := initTestRuntime(testSettings)
	return rt, rt.clock.(clockwork.FakeClock), rt.comms
}

func almStateRead(t *testing.T, c chan almStateMsg) (almStateMsg, error) {
	select {
	case e := <-c:
		return e, nil
	default:
		logCaller("Called from", 1)
		assert.Assert(t, false, "Nothing to read from alarm channel")
	}
	return almStateMsg{}, nil
}

func almStateReadAll(c chan almStateMsg) []almStateMsg {
	ret := make([]almStateMsg, 0)
	for true {
		select {
		case a := <-c:
			ret = append(ret, a)
		default:
			return ret
		}
	}
	return ret
}

func ledRead(t *testing.T, c chan ledEffect) (ledEffect, error) {
	select {
	case e := <-c:
		return e, nil
	default:
		logCaller("Called from", 1)
		assert.Assert(t, false, "Nothing to read from led channel")
	}
	return ledEffect{}, nil
}

func ledReadAll(c chan ledEffect) []ledEffect {
	ret := make([]ledEffect, 0)
	for true {
		select {
		case e := <-c:
			ret = append(ret, e)
		default:
			return ret
		}
	}
	return ret
}

func effectRead(t *testing.T, c chan displayEffect) (displayEffect, error) {
	select {
	case e := <-c:
		return e, nil
	default:
		logCaller("Called from", 1)
		assert.Assert(t, false, "Nothing to read from effect channel")
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

func effectReadAll(c chan displayEffect) []displayEffect {
	ret := make([]displayEffect, 0)
	for true {
		select {
		case e := <-c:
			ret = append(ret, e)
		default:
			return ret
		}
	}
	return nil
}

func unexpectedVal(channel string, v interface{}) string {
	return fmt.Sprintf("Got an unexpected value from %s: %v", channel, v)
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
		if clock.Now().Sub(start) >= d {
			keepClicking = false
		}
	}
}

func testQuit(rt runtimeConfig) {
	// close(rt.comms.quit)
	// rt.clock.(clockwork.FakeClock).Advance(time.Second)
}
