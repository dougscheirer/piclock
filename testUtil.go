package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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

func LogCaller(pc uintptr, file string, line int, ok bool) {
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

	log.Printf("Starting %s:%d %s", filepath.Base(file), line, fnName)
}

func testRuntime() runtimeConfig {
	// make rt for test
	LogCaller(runtime.Caller(1))
	return initTestRuntime(testSettings)
}

func almStateRead(t *testing.T, c chan almStateMsg) (almStateMsg, error) {
	select {
	case e := <-c:
		return e, nil
	default:
		assert.Assert(t, false, "Nothing to read from alarm channel")
	}
	return almStateMsg{}, nil
}

func almStateNoRead(t *testing.T, c chan almStateMsg) (almStateMsg, error) {
	select {
	case e := <-c:
		assert.Assert(t, e == almStateMsg{}, "Got an unexpected value on alarm channel")
	default:
	}
	return almStateMsg{}, nil
}

func ledRead(t *testing.T, c chan ledEffect) (ledEffect, error) {
	select {
	case e := <-c:
		return e, nil
	default:
		assert.Assert(t, false, "Nothing to read from led channel")
	}
	return ledEffect{}, nil
}

func ledNoRead(t *testing.T, c chan ledEffect) (ledEffect, error) {
	select {
	case e := <-c:
		assert.Assert(t, e == ledEffect{}, "Got an unexpected value from led channel")
	default:
	}
	return ledEffect{}, nil
}
