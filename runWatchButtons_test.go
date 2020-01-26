package main

import (
	"testing"
	"time"

	"github.com/stianeikeland/go-rpio"
	"gotest.tools/assert"
)

func TestButtonPress(t *testing.T) {
	rt, clock, comms := testRuntime()
	buttons := rt.buttons.(*noButtons)

	go runWatchButtons(rt)

	clock.BlockUntil(1)

	// set the main button to pressed
	btns := make(map[string]rpio.State)
	btns[sMainBtn] = rpio.Low
	buttons.set(btns)

	// advance clock, block and wait for a notification
	clock.Advance(dButtonSleep)
	clock.BlockUntil(1)

	effect, _ := effectRead(t, comms.effects)
	assert.Assert(t, effect.id == eMainButton, "Got non-main button effect")
	// should be buttonInfo
	btnInfo, _ := effect.val.(buttonInfo)
	assert.Assert(t, btnInfo.pressed == true)

	// let it go for a second so we get a duration
	testBlockDuration(clock, dButtonSleep, time.Second)

	// now release the button
	buttons.clear()
	// advance and re-check for clear
	clock.Advance(dButtonSleep)
	clock.BlockUntil(1)
	// close it
	close(comms.quit)
	clock.Advance(dButtonSleep)

	// we should get a second pressed message with a dutation of 1s
	effect, _ = effectRead(t, comms.effects)
	assert.Assert(t, effect.id == eMainButton, "Got non-main button effect")
	btnInfo, _ = effect.val.(buttonInfo)
	assert.Assert(t, btnInfo.pressed == true)
	assert.Assert(t, btnInfo.duration > 0)

	// we also should get a release message
	effect, _ = effectRead(t, comms.effects)
	assert.Assert(t, effect.id == eMainButton, "Got non-main button effect")
	btnInfo, _ = effect.val.(buttonInfo)
	assert.Assert(t, btnInfo.pressed == false)
}
