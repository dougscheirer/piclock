package main

import (
	"fmt"
	"log"
	"time"
)

func init() {
	wg.Add(1)
}

// return true if they look the same
func compareAlarms(alm1 alarm, alm2 alarm) bool {
	return (alm1.When == alm2.When && alm1.Effect == alm2.Effect &&
		alm1.Name == alm2.Name && alm1.Extra == alm2.Extra)
}

func runCheckAlarms(rt runtimeConfig) {
	defer wg.Done()
	defer func() {
		log.Println("exiting runCheckAlarm")
	}()

	settings := rt.settings
	alarms := make([]alarm, 0)
	comms := rt.comms

	var lastLogSecond = -1
	var curAlarm *alarm
	var buttonPressActed bool = false

	for true {
		// log.Printf("Read loop")
		// try reading from our channel
		select {
		case <-comms.quit:
			log.Println("quit from runCheckAlarm")
			return
		case stateMsg := <-comms.chkAlarms:
			switch stateMsg.ID {
			case msgLoaded:
				payload, _ := toLoadedPayload(stateMsg.val)
				alarms = payload.alarms
			case msgMainButton:
				info := stateMsg.val.(buttonInfo)
				if info.pressed {
					if buttonPressActed {
						log.Println("Ignore button hold")
					} else {
						log.Printf("Main button pressed: %dms", info.duration)
						// use curAlarm to figure out if we're doing an alarm
						// thing currently
						if curAlarm != nil {
							if curAlarm.started {
								comms.effects <- cancelAlarmMode(*curAlarm)
								buttonPressActed = true
							} else if curAlarm.countdown {
								comms.effects <- cancelAlarmMode(*curAlarm)
								curAlarm.started = true
								buttonPressActed = true
							}
						} else {
							// more than 5 seconds is "reload"
							if info.duration > 4*time.Second {
								comms.getAlarms <- reloadMessage()
								buttonPressActed = true
							}
						}
					}
				} else {
					buttonPressActed = false
					log.Printf("Main button released: %dms", info.duration/time.Millisecond)
				}
			}
		default:
			// continue
		}

		validAlarm := false
		// alarms come in sorted with soonest first
		for index := 0; index < len(alarms); index++ {
			if alarms[index].started {
				continue // skip processed alarms
			}

			// if alarms[index] != curAlarm, run some effects
			if curAlarm == nil || !compareAlarms(*curAlarm, alarms[index]) {
				curAlarm = &alarms[index]
				comms.effects <- printEffect("AL:", 1*time.Second)
				comms.effects <- printEffect(curAlarm.When.Format("15:04"), 2*time.Second)
				comms.effects <- printEffect(curAlarm.When.Format("01.02"), 2*time.Second)
				comms.effects <- printEffect(curAlarm.When.Format("2006"), 2*time.Second)
			}

			now := rt.clock.Now()
			duration := alarms[index].When.Sub(now)
			if lastLogSecond != now.Second() && now.Second()%30 == 0 {
				lastLogSecond = now.Second()
				log.Println(fmt.Sprintf("Time to next alarm: %ds (%ds to countdown)", duration/time.Second, (duration-settings.GetDuration(sCountdown))/time.Second))
			}

			// light the LED to show we have a pending alarm
			comms.leds <- ledOn(settings.GetInt(sLEDAlm))
			validAlarm = true

			if duration > 0 {
				// start a countdown?
				countdown := settings.GetDuration(sCountdown)
				if duration < countdown && !alarms[index].countdown {
					comms.effects <- setCountdownMode(alarms[0])
					alarms[index].countdown = true
				}
			} else {
				// Set alarm mode
				comms.effects <- setAlarmMode(alarms[index])
				// let getAlarms know we handled it (why?)
				comms.getAlarms <- handledMessage(alarms[index])
				alarms[index].started = true
			}
			break
		}
		if !validAlarm {
			comms.leds <- ledOff(settings.GetInt(sLEDAlm))
		}
		// take some time off
		rt.clock.Sleep(dAlarmSleep)
	}
}
