package main

import (
	"testing"

	"gotest.tools/assert"
)

func TestSetSecret(t *testing.T) {
	rt, clock, comms := testRuntime()
	testHandler := rt.configService.(*testConfigService)

	secret := rt.events.generateSecret(rt)
	// send in a secret
	comms.configSvc <- configSvcMsg{secret: secret}

	// launch the thread
	go runConfigService(rt)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	// make sure the secret got set
	assert.Equal(t, testHandler.handler.getSecret(), secret)
}

func TestAPIStatusGood(t *testing.T) {
	rt, clock, _ := testRuntime()
	testHandler := rt.configService.(*testConfigService)

	go runConfigService(rt)
	// wait for the init
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	status := testHandler.handler.getStatus()
	assert.Equal(t, status.Response, "OK")
	assert.Equal(t, status.Error, nil)
	assert.Equal(t, len(status.Alarms), 5)

	testQuit(rt)
}

func TestAPIStatusBad(t *testing.T) {
	rt, clock, _ := testRuntime()
	testHandler := rt.configService.(*testConfigService)
	tE := rt.events.(*testEvents)

	tE.setFails(1)

	go runConfigService(rt)
	// wait for the init
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	status := testHandler.handler.getStatus()
	assert.Equal(t, status.Response, "BAD")
	assert.Error(t, status.Error, "Bad fetch error")
	assert.Equal(t, len(status.Alarms), 0)

	testQuit(rt)
}

func TestAPIOauth(t *testing.T) {
	// might need to make an OAuth mocker?
	assert.Assert(t, false)
}
