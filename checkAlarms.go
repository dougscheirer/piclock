package main

import (
	"log"
	"net"
	"time"
)

func init() {
	wg.Add(1)
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

func startCheckAlarms(rt runtimeConfig) {
	rt.logger = &ThreadLogger{name: "Check alarms"}
	go runCheckAlarms(rt)
}

func runCheckAlarms(rt runtimeConfig) {
	defer wg.Done()
	defer func() {
		rt.logger.Println("exiting runCheckAlarm")
	}()

	settings := rt.settings
	comms := rt.comms
	buttonPressActed := false
	nextDoubleClick := time.Time{}

	// generate a new FSM
	state := newStateMachine(rt)
	for true {
		forceReport := false
		// rt.logger.Printf("Read loop")
		// try reading from our channel
		select {
		case <-comms.quit:
			rt.logger.Println("quit from runCheckAlarm")
			return
		case stateMsg := <-comms.chkAlarms:
			switch stateMsg.ID {
			case msgLoaded:
				payload, _ := toLoadedPayload(stateMsg.val)
				state.alarms = mergeAlarms(state.alarms, payload.alarms)
				forceReport = payload.report
				state.invalid = true
			case msgConfigError:
				state.setConfigError(stateMsg.val.(configError))
			case msgDoubleButton:
				// if there is a pending alarm ask to cancel
				info := stateMsg.val.(buttonInfo)
				if !info.pressed {
					continue
				}
				if info.duration > 0 || rt.clock.Now().Sub(nextDoubleClick) < 0 {
					rt.logger.Println("Ignoring duplicate doubleclick")
					continue
				}

				// if there is an alarm in the queue, attempt
				// to cancel it
				if state.isAlarmPlanned() {
					// this stays until it goes away with a single click
					state.startCancelPrompt()
				} else {
					nextDoubleClick = rt.clock.Now()
					if !state.hasConfigError() {
						nextDoubleClick = nextDoubleClick.Add(state.reportNextAlarm(true)) // this is essentially "none"
					}
					nextDoubleClick = nextDoubleClick.Add(state.showLoginInfo())
				}
			case msgLongButton:
				// reload on the 0th one only
				info := stateMsg.val.(buttonInfo)
				if info.pressed == true && info.duration == 0 {
					// manual reloads reset our existing list
					state.reset()
					comms.getAlarms <- reloadMessage()
				}
			case msgMainButton:
				info := stateMsg.val.(buttonInfo)
				rt.logger.Printf("Check alarms got main button msg: %v", info)
				if !info.pressed {
					buttonPressActed = false
					rt.logger.Printf("Main button released: %dms", info.duration/time.Millisecond)
					continue
				}
				if state.isCancelPrompting() {
					state.cancelPrompt()
					state.cancelNextAlarm()
					forceReport = true
				}
				if buttonPressActed {
					rt.logger.Println("Ignore button hold")
				} else {
					rt.logger.Printf("Main button pressed: %dms", info.duration)
					// only send it for the first press event
					if info.duration < time.Second {
						if state.cancelActiveAlarm() {
							forceReport = true
						}
					}
				}
			}
		default:
			// continue
		}

		// drive the state forward
		state.driveState(forceReport)

		if !state.hasNextAlarm() {
			comms.leds <- ledOff(settings.GetInt(sLEDAlm))
		}

		// take some time off
		// rt.logger.Printf("Sleep")
		rt.clock.Sleep(dAlarmSleep)
	}
}
