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

func runCheckAlarms(rt runtimeConfig) {
	defer wg.Done()
	defer func() {
		log.Println("exiting runCheckAlarm")
	}()

	settings := rt.settings
	comms := rt.comms
	buttonPressActed := false

	// generate a new FSM
	state := newStateMachine(rt)
	for true {
		forceReport := false
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
				state.mergeAlarms(payload.alarms)
				forceReport = payload.report
			case msgConfigError:
				state.setConfigError(stateMsg.val.(configError))
			case msgDoubleButton:
				// if there is a pending alarm ask to cancel
				info := stateMsg.val.(buttonInfo)
				if !info.pressed {
					continue
				}
				if info.duration > 0 {
					log.Println("Ignoring duplicate doubleclick")
					continue
				}

				// if there is an alarm in the queue, attempt
				// to cancel it
				if state.isAlarmPlanned() {
					state.startCancelPrompt()
				} else {
					if !state.hasConfigError() {
						state.reportNextAlarm(true) // this is essentially "none"
					}
					state.showLoginInfo()
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
				log.Printf("Check alarms got main button msg: %v", info)
				if !info.pressed {
					buttonPressActed = false
					log.Printf("Main button released: %dms", info.duration/time.Millisecond)
					continue
				}
				if state.isCancelPrompting() {
					state.cancelPrompt()
					state.cancelNextAlarm()
					forceReport = true
				}
				if buttonPressActed {
					log.Println("Ignore button hold")
				} else {
					log.Printf("Main button pressed: %dms", info.duration)
					// only send it for the first press event
					if info.duration < time.Second {
						state.cancelActiveAlarm()
						forceReport = true
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
		// log.Printf("Sleep")
		rt.clock.Sleep(dAlarmSleep)
	}
}
