package main

import (
	"testing"
	"time"

	"github.com/stianeikeland/go-rpio"
	"gotest.tools/assert"
)

func TestLongButtonPressPulldown(t *testing.T) {
	DoTestButtonPress(false, sLongBtn, eLongButton, t)
}

func TestLongTButtonPressPullup(t *testing.T) {
	DoTestButtonPress(true, sLongBtn, eLongButton, t)
}

func TestDoubleButtonPressPulldown(t *testing.T) {
	DoTestButtonPress(false, sDblBtn, eDoubleButton, t)
}

func TestDoubleButtonPressPullup(t *testing.T) {
	DoTestButtonPress(true, sDblBtn, eDoubleButton, t)
}

func TestButtonPressPulldown(t *testing.T) {
	DoTestButtonPress(false, sMainBtn, eMainButton, t)
}

func TestButtonPressPullup(t *testing.T) {
	DoTestButtonPress(true, sMainBtn, eMainButton, t)
}

func DoTestButtonPress(pullup bool, btnName string, eid int, t *testing.T) {
	rt, clock, comms := testRuntime()
	buttons := rt.buttons.(*noButtons)

	// change the pullup setting, only set the target to the right direction
	names := rt.settings.GetAllButtonNames()
	for i := range names {
		bm := rt.settings.GetButtonMap(names[i])
		bm.pullup = !pullup
		rt.settings.settings[names[i]] = bm
	}
	// set the target
	bm := rt.settings.GetButtonMap(btnName)
	bm.pullup = pullup
	rt.settings.settings[btnName] = bm

	// start the watcher
	go runWatchButtons(rt)

	clock.BlockUntil(1)

	// toggle the target button to "pressed"
	slist, _ := buttons.readButtons(rt)
	if pullup {
		slist[btnName] = rpio.Low
	} else {
		slist[btnName] = rpio.High
	}
	buttons.setStates(slist)

	// advance clock, block and wait for a notification
	clock.Advance(dButtonSleep)
	clock.BlockUntil(1)

	effect, _ := effectRead(t, comms.effects)
	assert.Equal(t, effect.id, eid, "Got wrong button effect")
	// should be buttonInfo
	btnInfo, _ := effect.val.(buttonInfo)
	assert.Equal(t, btnInfo.pressed, true)
	effectNoRead(t, comms.effects)

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

	// we should get a second pressed message with a duration of 1s
	effect, _ = effectRead(t, comms.effects)
	assert.Equal(t, effect.id, eid, "Got wrong button effect")
	btnInfo, _ = effect.val.(buttonInfo)
	assert.Equal(t, btnInfo.pressed, true)
	// only the main button gets a duration, the others are really single events
	if btnName == sMainBtn {
		assert.Assert(t, btnInfo.duration > 0)
	} else {
		assert.Assert(t, btnInfo.duration == 0)
	}

	// we also should get a release message
	effect, _ = effectRead(t, comms.effects)
	assert.Equal(t, effect.id, eid, "Got wrong button effect")
	btnInfo, _ = effect.val.(buttonInfo)
	assert.Equal(t, btnInfo.pressed, false)

	// done
	testQuit(rt)
}
