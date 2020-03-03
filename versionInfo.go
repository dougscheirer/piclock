package main

import "fmt"

const versionSHA string = "unknown"

func getVersionInfo() string {
	return versionSHA
}

func showVersionInfo() {
	fmt.Printf("Version %s", versionSHA)
}
