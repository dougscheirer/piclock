package main

import "time"

func init() {
	wg.Add(1)
}

func startNTPWatcher(rt runtimeConfig) {
	rt.logger = &ThreadLogger{name: "NTPWatcher"}
	go runNTPWatcher(rt)
}

func runNTPWatcher(rt runtimeConfig) {
	defer wg.Done()
	defer func() {
		rt.logger.Println("Exiting runNTPWatcher")
	}()

	for true {
		select {
		case <-rt.comms.quit:
			rt.logger.Println("quit from runCheckAlarm")
			return
		default:
		}
		ipTime := rt.ntpCheck.getIPDateTime(rt)
		diff := rt.clock.Now().Sub(ipTime)

		// is our clock more than 5m off?
		if diff > time.Minute*5 || diff < time.Minute*-5 {
			// print a message, also error flag
			rt.comms.effects <- printRollingEffect(sNeedSync, 5*time.Second)
		}
		rt.clock.Sleep(dNTPCheckSleep)
	}
}
