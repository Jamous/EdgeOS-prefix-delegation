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
		createLogs(fmt.Sprintf("getConfig could not get edgeroute config. Command: %s Error: %s", configPath, err), false)
		os.Exit(1)
	}

	//Convert into a slice of configs and return
	config := strings.Split(string(dconfig), "\n")

	return config
}

// sets system config as stirng. Input example "set system hostname myhost\n"
func setConfig(configCommands []string) {
	// Create the script to set commands
	commands := createScript(configCommands)

	// Define the file path and the string to write
	configFilePath := "/tmp/pd_config.sh"

	// Write script to disk
	writeScript(commands, configFilePath)

	//Execute script
	executeScript(configFilePath)

	//Remove script
	removeScript(configFilePath)
}

// Create the script to set commands
func createScript(configCommands []string) string {
	//setup command script
	commands := "#!/bin/vbash\n"
	commands += "runcfg=/opt/vyatta/sbin/vyatta-cfg-cmd-wrapper\n"
	commands += "$runcfg begin\n"

	//Add in commands
	for _, command := range configCommands {
		commands += fmt.Sprintf("$runcfg %s", command)
	}

	//Close script
	commands += "$runcfg commit\n"
	commands += "$runcfg save\n"
	commands += "$runcfg end\n"

	//Return commands
	return commands
}

// Write script to disk
func writeScript(commands string, configFilePath string) {
	// Write the string to the file
	err := os.WriteFile(configFilePath, []byte(commands), 0777)
	if err != nil {
		createLogs(fmt.Sprintf("writeScript could not created file %s, error %s", configFilePath, err), false)
		os.Exit(1)
	}
	createLogs(fmt.Sprintf("writeScript created file %s", configFilePath), true)
}

// Execute script
func executeScript(configFilePath string) {
	// Create the command to execute the script
	cmd := exec.Command("bash", configFilePath)

	// Run the command and capture any errors
	err := cmd.Run()
	if err != nil {
		createLogs(fmt.Sprintf("executeScript could not execute script file %s, error %s", configFilePath, err), false)
		os.Exit(1)
	}
	createLogs(fmt.Sprintf("executeScript executed file %s", configFilePath), true)
}

// Remove script
func removeScript(configFilePath string) {
	// Remove the file
	err := os.Remove(configFilePath)
	if err != nil {
		createLogs(fmt.Sprintf("removeScript could not remove script file %s, error %s", configFilePath, err), false)
		os.Exit(1)
	}
	createLogs(fmt.Sprintf("removeScript removed file %s", configFilePath), true)
}
