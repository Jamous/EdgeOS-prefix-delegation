package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Return system config as stirng. Input example "run show configuration commands \"service dhcpv6-server\"\n". Returns each line as a string
func getConfig(configPath string) []string {
	commands := "#!/bin/vbash\n"
	commands += "source /opt/vyatta/etc/functions/script-template\n"
	commands += configPath
	commands += "exit\n"

	// Execute the script using vbash
	cmd := exec.Command("/bin/vbash", "-c", commands)
	dconfig, err := cmd.CombinedOutput()
	if err != nil {
		createLogs(fmt.Sprintf("getConfig could not get edgeroute config. Error: %s Config: %s", err, commands), false)
		os.Exit(1)
	}

	//Convert into a slice of configs and return
	config := strings.Split(string(dconfig), "\n")

	return config
}

// sets system config as stirng. Input example "set system hostname myhost\n"
func setConfig(configCommands string) {
	commands := "#!/bin/vbash\n"
	commands += "source /opt/vyatta/etc/functions/script-template\n"
	commands += "configure\n"
	commands += configCommands
	commands += "exit\n"
	commands += "exit\n"

	// Execute the script using vbash
	cmd := exec.Command("/bin/vbash", "-c", commands)
	err := cmd.Run()
	if err != nil {
		createLogs(fmt.Sprintf("setConfig could not update edgeroute config. Error: %s Config: %s", err, commands), false)
		os.Exit(1)
	}
}
