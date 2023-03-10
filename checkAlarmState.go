package main

import (
	"fmt"
	"time"
)

const (
	modeDefault = iota
	modeCancelStarted
	modeCancelled
)

type cancelMode struct {
	mode        int
	startCancel time.Time
}

type rca struct {
	rt          runtimeConfig
	mode        cancelMode
	alarms      []alarm
	nextAlarm   *alarm
	activeAlarm *alarm
	lastLog     int
	cfgError    configError
	cancelPrint chan bool
	invalid     bool
}

func (state *rca) showLoginInfo() time.Duration {
	e := state.rt.comms.effects
	// show a secret code and our IP address
	e <- printRollingEffect("secret", dRollingPrint)
	e <- printEffect(state.cfgError.secret, dPrintDuration)
	e <- printEffect("IP:  ", dPrintDuration)
	e <- printRollingEffect(GetOutboundIP().String(), dRollingPrint)
	return calcRolling("secret") + dPrintDuration + dPrintDuration + calcRolling(GetOutboundIP().String())
}

func mergeAlarms(curAlarms []alarm, newAlarms []alarm) []alarm {
	if len(curAlarms) == 0 {
		return newAlarms
	}

	// the resultant list consists of all of the alarms
	// in sequence order.  for any alarms that "look"
	// the same, we leave the handled/countdown states
	// as they are in our current list

	// when you rescedule an alarm it keeps the same
	// ID.  we will walk through the newAlarm list, remove
	// things that match from curAlarms and preserve the
	// handled attribute

	// turn curAlarms into a map based on ID
	curMap := make(map[string]alarm, 0)
	for i := range curAlarms {
		curMap[curAlarms[i].ID] = curAlarms[i]
	}
	result := make([]alarm, 0)
	for i := range newAlarms {
		alm := newAlarms[i]
		curAlm, exists := curMap[alm.ID]
		// if the ID matches and the time is the same
		// copy the handled bits
		if exists && alm.When == curAlm.When {
			alm.countdown = curAlm.countdown
			alm.started = curAlm.started
		}
		result = append(result, alm)
	}
	return result
}

func (state *rca) isAlarmPlanned() bool {
	return state.nextAlarm != nil
}

func newStateMachine(rt runtimeConfig) *rca {
	return &rca{
		alarms:      make([]alarm, 0),
		mode:        cancelMode{mode: modeDefault},
		nextAlarm:   nil,
		rt:          rt,
		activeAlarm: nil,
		lastLog:     -1,
		cancelPrint: make(chan bool, 10),
		invalid:     true,
	}
}

func (state *rca) clearAlarms() {
	state.alarms = make([]alarm, 0)
	state.nextAlarm = nil
	state.invalid = true
	// maybe? we may lose track if a reload comes during an active alarm
	state.activeAlarm = nil
}

func (state *rca) hasNextAlarm() bool {
	return state.nextAlarm != nil
}

func (state *rca) cancelMessages() {
	state.cancelPrint <- true
	close(state.cancelPrint)
	state.cancelPrint = make(chan bool, 10)
}

func (state *rca) driveState(forceReport bool) {
	if state.mode.mode == modeCancelStarted && state.rt.clock.Now().Sub(state.mode.startCancel) >= dCancelTimeout {
		state.rt.logger.Println("Cancel timed out")
		// we reuse cancelPrint for multiple messages, so close and reopen the channel
		// to ensure that all messages are cancelled
		state.cancelMessages()
		state.mode.mode = modeDefault
		state.reportNextAlarm(forceReport)
		return
	}

	// TODO: this is two state functions, "find next alarm"
	//       and "activate next alarm"
	comms := state.rt.comms
	settings := state.rt.settings
	now := state.rt.clock.Now()
	nowSec := now.Second()

	if state.invalid {
		newNextAlarm := state.findNextAlarm()
		// TODO: use the compare function?
		if !state.compareAlarms(newNextAlarm, state.nextAlarm) || forceReport {
			state.nextAlarm = newNextAlarm
			// only report when there is no active alarm
			if state.activeAlarm == nil {
				state.reportNextAlarm(forceReport)
			}
		}
		state.invalid = false
	}

	if state.nextAlarm == nil {
		return
	}

	duration := state.nextAlarm.When.Sub(now)

	if state.lastLog != nowSec && nowSec%30 == 0 {
		state.lastLog = nowSec
		state.rt.logger.Println(fmt.Sprintf("Time to next alarm: %ds (%ds to countdown)", duration/time.Second, (duration-settings.GetDuration(sCountdown))/time.Second))
	}

	// light the LED to show we have a pending alarm
	state.rt.comms.leds <- ledOn(settings.GetInt(sLEDAlm))

	// check if we're close
	if duration > 0 {
		// start a countdown?
		countdown := settings.GetDuration(sCountdown)
		if duration < countdown && !state.nextAlarm.countdown {
			// remember this one for later
			state.activeAlarm = state.nextAlarm
			comms.effects <- setCountdownMode(state.alarms[0])
			state.nextAlarm.countdown = true
		}
	} else if state.activeAlarm == nil || !state.activeAlarm.started {
		// Set alarm mode
		// remember this one for later
		state.activeAlarm = state.nextAlarm
		comms.effects <- setAlarmMode(*state.nextAlarm)
		// let getAlarms know we handled it (why?)
		comms.getAlarms <- handledMessage(*state.nextAlarm)
		state.nextAlarm.started = true
	}
}

func (state *rca) findNextAlarm() *alarm {
	// alarms come in sorted with soonest first
	for index := 0; index < len(state.alarms); index++ {
		if state.alarms[index].started {
			continue // skip processed alarms
		}
		return &state.alarms[index]
	}
	return nil
}

func (state *rca) isCancelPrompting() bool {
	return state.mode.mode == modeCancelStarted
}

func (state *rca) cancelNextAlarm() {
	state.mode.mode = modeDefault
	state.invalid = true

	if state.nextAlarm == nil {
		return
	}

	state.nextAlarm.started = true
}

func (state *rca) startCancelPrompt() {
	state.rt.comms.effects <- printCancelableRollingEffect(sCancel, dRollingPrint, state.cancelPrint)
	state.rt.comms.effects <- printCancelableEffect(sYorN, 0, state.cancelPrint) // no duration is until cancelled
	state.mode.mode = modeCancelStarted
	// do the math: Y : n should be displayed for n secs, add time to print the rolling effect right before it
	now := state.rt.clock.Now()
	offset := calcRolling(sCancel)
	state.mode.startCancel = now.Add(offset)
	state.rt.logger.Printf("Start : %s", now.Format("2006-01-02T15:04:05.999999"))
	state.rt.logger.Printf("YorN  : %s", state.mode.startCancel.Format("2006-01-02T15:04:05.999999"))
}

func (state *rca) cancelPrompt() {
	state.rt.logger.Println("Cancel next alarm")
	// make sure to queue up the next print first
	state.rt.comms.effects <- printRollingEffect("-- cancelled --", dRollingPrint)
	state.cancelMessages()
}

func (state *rca) setConfigError(err configError) {
	state.cfgError = err
	// TODO: set other state info?
}

func (state *rca) hasConfigError() bool {
	return state.cfgError.err
}

func (state *rca) reportNextAlarm(force bool) time.Duration {
	comms := state.rt.comms
	alm := state.nextAlarm

	var duration time.Duration = 0

	if alm != nil {
		// calculate days/hours/minutes
		now := state.rt.clock.Now()
		diff := alm.When.Sub(now)
		days := int(diff.Hours() / 24)
		diff = diff - time.Duration(days*24)*time.Hour
		hours := int(diff.Hours())
		diff = diff - time.Duration(hours)*time.Hour
		// more than 7 days, print the date
		if days > 7 {
			comms.effects <- printRollingEffect(sNextAL, dRollingPrint)
			duration += calcRolling(sNextAL)
			// date, then time
			effect := alm.When.Format("01.02 2006")
			comms.effects <- printRollingEffect(effect, dRollingPrint)
			duration += calcRolling(effect)
			comms.effects <- printEffect(sAt, dPrintDuration)
			duration += dPrintDuration
			comms.effects <- printEffect(alm.When.Format("15:04"), dPrintDuration)
			duration += dPrintDuration
		} else {
			comms.effects <- printRollingEffect(sNextALIn, dRollingPrint)
			duration += calcRolling(sNextALIn)
			if days > 0 {
				comms.effects <- printEffect(fmt.Sprintf("%dd", days), dPrintDuration)
				duration += dPrintDuration
			}
			comms.effects <- printEffect(fmt.Sprintf("%2d:%02d", hours, int(diff.Minutes())), dPrintDuration)
			duration += dPrintDuration
		}
	} else {
		// only print "none" when specifically asked?
		// if force {
		comms.effects <- printEffect("none", dPrintBriefDuration)
		duration += dPrintBriefDuration
		// }
	}
	return duration
}

// return true if they look the same
func (state *rca) compareAlarms(alm1 *alarm, alm2 *alarm) bool {
	if alm1 == nil && alm2 == nil {
		return true
	}
	if (alm1 != nil && alm2 == nil) || (alm1 == nil && alm2 != nil) {
		return false
	}
	return (alm1.ID == alm2.ID && alm1.When == alm2.When && alm1.Effect == alm2.Effect &&
		alm1.Name == alm2.Name && alm1.Extra == alm2.Extra)
}

func (state *rca) reset() {
	state = newStateMachine(state.rt)
}

func (state *rca) cancelActiveAlarm() bool {
	if state.activeAlarm == nil {
		return false
	}
	state.activeAlarm.started = true
	state.activeAlarm = nil
	state.rt.comms.effects <- cancelAlarmMode()
	state.invalid = true
	return true
}

func (state *rca) isActiveAlarm() bool {
	return state.activeAlarm != nil
}
