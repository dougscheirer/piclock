package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
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

func showNextAlarm(rt runtimeConfig, alm *alarm) {
	if alm != nil {
		rt.comms.effects <- printEffect("AL:", 1*time.Second)
		rt.comms.effects <- printEffect(alm.When.Format("15:04"), 2*time.Second)
		rt.comms.effects <- printEffect(alm.When.Format("01.02"), 2*time.Second)
		rt.comms.effects <- printEffect(alm.When.Format("2006"), 2*time.Second)
	} else {
		rt.comms.effects <- printEffect("none", 1*time.Second)
	}
}

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func showLoginInfo(rt runtimeConfig) {
	// show a secret code and our IP address
	// TODO: remember this for more than an instant
	s1 := rand.NewSource(rt.clock.Now().UnixNano())
	r1 := rand.New(s1)
	secret := r1.Intn(0xFFFF)
	rt.comms.effects <- printEffect("sec", 3*time.Second)
	rt.comms.effects <- printEffect(fmt.Sprintf("%04x", secret), 3*time.Second)
	rt.comms.effects <- printEffect("IP:", 3*time.Second)
	parts := strings.Split(GetOutboundIP().String(), ".")
	for i := range parts {
		rt.comms.effects <- printEffect(parts[i], 3*time.Second)
	}
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
	var curAlarm *alarm // the alarm we are watching
	var nowAlarm *alarm // the alarm that is running now
	var buttonPressActed bool = false
	var configError bool = false

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
				if payload.report {
					// poor side-effect, report by resetting "curAlarm"
					curAlarm = nil
				}
			case msgConfgError:
				configError = stateMsg.val.(bool)
			case msgDoubleButton:
				// show next alarm on the 0th one only
				info := stateMsg.val.(buttonInfo)
				if info.pressed == true && info.duration == 0 {
					// are we in a bad state?
					if configError {
						showLoginInfo(rt)
					} else {
						showNextAlarm(rt, curAlarm)
					}
				}
			case msgLongButton:
				// reload on the 0th one only
				info := stateMsg.val.(buttonInfo)
				if info.pressed == true && info.duration == 0 {
					comms.getAlarms <- reloadMessage()
				}
			case msgMainButton:
				// TODO: cancel on the 0th one only
				info := stateMsg.val.(buttonInfo)
				if info.pressed {
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
