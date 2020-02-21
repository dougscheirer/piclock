package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

func init() {
	wg.Add(1)
}

// return true if they look the same
func compareAlarms(alm1 alarm, alm2 alarm) bool {
	return (alm1.ID == alm2.ID && alm1.When == alm2.When && alm1.Effect == alm2.Effect &&
		alm1.Name == alm2.Name && alm1.Extra == alm2.Extra)
}

func showNextAlarm(rt runtimeConfig, alm *alarm) {
	if alm != nil {
		rt.comms.effects <- printRollingEffect(dNextAL, dRollingPrint)
		// calculate days/hours/minutes
		now := rt.clock.Now()
		diff := alm.When.Sub(now)
		days := int(diff.Hours() / 24)
		diff = diff - time.Duration(days*24)*time.Hour
		hours := int(diff.Hours())
		diff = diff - time.Duration(hours)*time.Hour
		if days > 999 {
			rt.comms.effects <- printRollingEffect(fmt.Sprintf("%dd", days), dRollingPrint)
		} else if days > 0 {
			rt.comms.effects <- printEffect(fmt.Sprintf("%dd", days), 3*time.Second)
		}
		rt.comms.effects <- printEffect(fmt.Sprintf("%2d:%02d", hours, int(diff.Minutes())), 3*time.Second)
	} else {
		rt.comms.effects <- printEffect("none", 1*time.Second)
	}
}

// GetOutboundIP - Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func showLoginInfo(rt runtimeConfig, secret string) {
	// show a secret code and our IP address
	rt.comms.effects <- printRollingEffect("secret", dRollingPrint)
	rt.comms.effects <- printEffect(secret, 3*time.Second)
	rt.comms.effects <- printEffect("IP:  ", 3*time.Second)
	rt.comms.effects <- printRollingEffect(GetOutboundIP().String(), dRollingPrint)
}

func mergeAlarms(rt runtimeConfig, oldAlarms []alarm, newAlarms []alarm) []alarm {
	return newAlarms
}

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
	mode        cancelMode
	alarms      []alarm
	nextAlarm   *alarm
	activeAlarm *alarm
	lastLog
}

const cancelTimeout time.Duration = 5 * time.Second

func driveCancelMode(rt runtimeConfig, rca rca) rca {
	if rca.mode.mode != modeCancelStarted {
		return rca
	}

	if rt.clock.Now().Sub(rca.mode.startCancel) >= cancelTimeout {
		showNextAlarm(rt, rca.nextAlarm)
	}
}

func runCheckAlarms(rt runtimeConfig) {
	defer wg.Done()
	defer func() {
		log.Println("exiting runCheckAlarm")
	}()

	settings := rt.settings
	comms := rt.comms

	// we maintain some data
	//   alarm list: cannonical list of old and new alarms
	//							 on manual reload we reset to nothing
	//							 on timed reload we merge the two to keep modified status
	//   activeAlarm: an alarm that is current firing or counting down
	//   nextAlarm:   the next alarm in the queue that is not cancelled.  may
	//								be the same as activeAlarm
	//   buttonPressActed: a state variable to indicate that we handled the previous button press (?)
	//   mode: if we're cancelling, the state of the cancelling FSM
	//   lastLog: for logging, we periodically output time until the next alarm
	//   cfgErr: if we got an error msg, that
	//   cancelPrint: a channel for cancelling the print msgs

	// TODO: this should probably be in a FSM
	var state rca = rca{alarms: make([]alarm, 0)}
	var lastLog = -1
	var buttonPressActed bool = false
	var cfgErr configError
	var cancel cancelMode

	cancelPrint := make(chan bool, 10)

	for true {
		cancelMode = driveCancelMode(rt, cancel)

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
				alarms = mergeAlarms(rt, alarms, payload.alarms)
				if payload.report {
					// poor side-effect, report by resetting "curAlarm"
					curAlarm = nil
				}
			case msgConfigError:
				cfgErr = stateMsg.val.(configError)
			case msgDoubleButton:
				// if there is a pending alarm ask to cancel
				info := stateMsg.val.(buttonInfo)
				if info.pressed == true {
					if info.duration > 0 {
						log.Println("Ignoring duplicate doubleclick")
					} else {
						if curAlarm != nil {
							comms.effects <- printCancelableRollingEffect("cancel", dRollingPrint, cancelPrint)
							comms.effects <- printCancelableEffect("Y : n", 3*time.Second, cancelPrint)
							cancelMode = rt.clock.Now()
						} else {
							// are we in a bad state?
							if cfgErr.err {
								showLoginInfo(rt, cfgErr.secret)
							} else {
								showNextAlarm(rt, curAlarm)
								showLoginInfo(rt, cfgErr.secret)
							}
						}
					}
				}
			case msgLongButton:
				// reload on the 0th one only
				info := stateMsg.val.(buttonInfo)
				if info.pressed == true && info.duration == 0 {
					// manual reloads reset our existing list
					alarms = nil
					curAlarm = nil
					nowAlarm = nil
					comms.getAlarms <- reloadMessage()
				}
			case msgMainButton:
				info := stateMsg.val.(buttonInfo)
				log.Printf("Check alarms got main button msg: %v", info)
				if info.pressed {
					if cancelMode != noTime {
						log.Println("Cancel next alarm")
						cancelPrint <- true
						comms.effects <- printRollingEffect("-- cancelled --", dRollingPrint)
						curAlarm.started = true
						nowAlarm = nil
						cancelMode = noTime
					}
					if buttonPressActed {
						log.Println("Ignore button hold")
					} else {
						log.Printf("Main button pressed: %dms", info.duration)
						// only send it for the first press event
						if info.duration < time.Second {
							comms.effects <- cancelAlarmMode()
							if nowAlarm != nil {
								nowAlarm.started = true
							}
							nowAlarm = nil
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
				showNextAlarm(rt, curAlarm)
			}

			now := rt.clock.Now()
			duration := alarms[index].When.Sub(now)
			if lastLog != now.Second() && now.Second()%30 == 0 {
				lastLog = now.Second()
				log.Println(fmt.Sprintf("Time to next alarm: %ds (%ds to countdown)", duration/time.Second, (duration-settings.GetDuration(sCountdown))/time.Second))
			}

			// light the LED to show we have a pending alarm
			comms.leds <- ledOn(settings.GetInt(sLEDAlm))
			validAlarm = true

			if duration > 0 {
				// start a countdown?
				countdown := settings.GetDuration(sCountdown)
				if duration < countdown && !alarms[index].countdown {
					// remember this one for later
					nowAlarm = curAlarm
					comms.effects <- setCountdownMode(alarms[0])
					alarms[index].countdown = true
				}
			} else {
				// Set alarm mode
				// remember this one for later
				nowAlarm = curAlarm
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
