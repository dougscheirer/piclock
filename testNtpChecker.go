package main

import "time"

type testNtpChecker struct {
	curtime time.Time
}

func (ntp *testNtpChecker) getIPDateTime(rt runtimeConfig) time.Time {
	return ntp.curtime
}
