// utility functions
package main

type CommChannels struct {
	quit    chan struct{}
	alarms  chan CheckMsg
	effects chan Effect
	loader  chan LoaderMsg
}

func initCommChannels() CommChannels {
	quit := make(chan struct{}, 1)
	alarmChannel := make(chan CheckMsg, 1)
	effectChannel := make(chan Effect, 1)
	loaderChannel := make(chan LoaderMsg, 1)

	// wait on our workers:
	// alarm fetcher
	// clock runner (effects)
	// alarm checker
	// button checker
	wg.Add(4)

	return CommChannels{quit: quit, alarms: alarmChannel, effects: effectChannel, loader: loaderChannel}
}
