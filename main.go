package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"log/syslog"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// evil global variables
var debugFlag bool
var logger *log.Logger
var version string = "0.1"

// Struct to hold subnet details
type pdnet struct {
	Subnet *net.IPNet
	Start  netip.Addr
	Stop   netip.Addr
}

// Main function
func main() {
	//Set standard exitcode.
	exitCode := 0

	// Setup logger, log to syslog at /var/log/messages. Program name is dhcpv6-pd
	sysLogger := setupLogger()
	defer sysLogger.Close()

	//Setup defer to cleanup program
	defer func() {
		//Catch unrecoverable errors, log to syslog at /var/log/messages.
		if r := recover(); r != nil {
			logger.Printf("dhcpv6-pd failed with an error: %s", r)
		}

		//Custom exit code. 0 for okay, 1 for fatal, 2 for user logged in.
		os.Exit(exitCode)
	}()

	// Parse user input
	bypassCheck := parseInput()

	//Check if a user is logged in. If no user is logged in, or if bypass is checked, continue
	loggedIn := false
	if !bypassCheck {
		loggedIn = userLoggedIn()
	}

	//If no user is logged in, procede.
	if !loggedIn {
		// Debug log
		createLogs("User is not logged in, running program.", true)

		//Get the current DHCHv6-PD leases from /config/dhcpdv6.leases. Return as a chunked list of the log, currentLeases.
		leaseEntries := leaseFileParser()

		//Parse leaseEntries and find matches between leases. Returns routeList
		routeList := getCurrentLeases(leaseEntries)

		//Get current ipv6 routes
		currentRoutes := getCurrentRoutes()

		//Compare routes and leases, get back list of routes to remove
		removeRoutes, addRoutes := compareRoutes(currentRoutes, routeList)

		//Issue updates to edgerouter
		updateEdgerouter(removeRoutes, addRoutes)

		//Log changes
		if len(removeRoutes) > 0 || len(addRoutes) > 0 {
			logChanges(removeRoutes, addRoutes)
		}
	} else {
		//Create debug log
		createLogs("User is logged in, terminating program.", true)
		exitCode = 2
	}
}

// Parse user input
func parseInput() bool {
	//Flags
	bypassCheck := flag.Bool("b", false, "Bypass check to see if a user is active on the router.")
	debugflag := flag.Bool("d", false, "Enable debug logs")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("EdgeOS prefix delegation. https://github.com/Jamous/EdgeOS-prefix-delegation \nVersion %s", version))
		fmt.Fprintln(os.Stderr, "Usage:")
		flag.PrintDefaults()
	}

	flag.Parse()
	debugFlag = *debugflag

	return *bypassCheck
}

// Setup logger
func setupLogger() *syslog.Writer {
	// Connect to the system logger
	sysLogger, err := syslog.New(syslog.LOG_NOTICE|syslog.LOG_LOCAL7, "dhcpv6-pd")
	if err != nil {
		os.Exit(1)
	}

	// Create a logger that writes to syslog
	logger = log.New(sysLogger, "", 0)

	return sysLogger
}

// Process logs, wrapper for logger to determin if debug is set or not.
func createLogs(log string, debug bool) {
	//If this is a standard log, write
	if !debug {
		logger.Print(log)
	} else {
		//If this is a debug log, check if debug is enabeled. If so, write
		if debugFlag {
			logger.Print(log)
		}
	}
}

// Checks if someone is currently using the router. Do not run while in use (ssh or web)
func userLoggedIn() bool {
	//Get the ports used for web access. Accomodates custom ports being used.
	cmd := exec.Command("sh", "-c", "/opt/vyatta/sbin/my_cli_shell_api showConfig service gui | grep port | awk '{print $2}'")
	output, err := cmd.CombinedOutput()
	if err != nil {
		createLogs(fmt.Sprintf("userLoggedIn get ports: %s", err), false)
		os.Exit(1)
	}
	ports := strings.Split(strings.TrimSpace(string(output)), "\n")

	//Check if the web interface is in use, if it is return true
	cmd2 := exec.Command("sh", "-c", "netstat -tn | awk '{print $4}'")
	output2, err := cmd2.CombinedOutput()
	if err != nil {
		createLogs(fmt.Sprintf("userLoggedIn web check error: %s", err), false)
		os.Exit(1)
	}
	netstat := string(output2)

	//Search netstat for ports, if the port is found return true
	for _, port := range ports {
		if strings.Contains(netstat, port) {
			return true
		}
	}

	//Check if there is a tty session open, if so return true, otherwise return false
	cmd3 := exec.Command("sh", "-c", "who")
	output3, err := cmd3.CombinedOutput()
	if err != nil {
		createLogs(fmt.Sprintf("userLoggedIn tty check: %s", err), false)
		os.Exit(1)
	}
	return len(output3) > 0
}

// Reads in lease file, chuncks up leaes and returns as a slice
func leaseFileParser() []string {
	leaseFile := "/config/dhcpdv6.leases"

	//Open file, returns pointer
	leases, err := os.ReadFile(leaseFile)

	//Check if file was read, if not raise error
	if err != nil {
		createLogs(fmt.Sprintf("leaseFileParser could not read file %s: %s", leaseFile, err), false)
		os.Exit(1)
	}

	//Convert leaseFile to string
	leaseString := string(leases)

	//Varibels
	var leaseEntries []string //Each entry is converted into its own object
	var currentEntrie string  //current leaseEntry

	//Iterate through the leaseString, convert each large entry into its own leaseEntrie, break up based on empty lines
	scanner := bufio.NewScanner(strings.NewReader(leaseString))
	for scanner.Scan() {
		line := scanner.Text()

		//If line is empty, move currentEntrie to leaseEntries and start over
		if strings.TrimSpace(line) == "" {
			//Verify entry contains at least one {
			if strings.Contains(currentEntrie, "{") {
				leaseEntries = append(leaseEntries, currentEntrie)
			}

			//Reset currentEntrie
			currentEntrie = ""

		} else {
			//Add line to currentEntrie
			currentEntrie += line
		}
	}

	return leaseEntries
}

// Gets the current ipv6-pd leases from leaseEntries
func getCurrentLeases(leaseEntries []string) map[string]string {
	//Variabels
	ianaPairs := make(map[string]string) //iana pairs [router ID]WAN address
	iapdPairs := make(map[string]string) //iapd pairs [router ID]Subnet
	routeList := make(map[string]string) //routes in the lease document [subnet]destination. Ex [2001:db8:1::/34]2001:db8::2

	//Iterate through each lease entry
	for _, entry := range leaseEntries {
		//If the entry contains ia-na, find the router ID and router address.
		if strings.Contains(entry, "ia-na") {
			iana, iaaddr := findIANA(entry)
			ianaPairs[iana] = iaaddr

		} else if strings.Contains(entry, "ia-pd") {
			//If the entry contains ia-pd, find the router ID and IPv6 subnet that was handed out.
			iapd, iaprefix := findIAPD(entry)
			iapdPairs[iapd] = iaprefix
		}
	}

	//Find matches between ianaPairs, iapdPairs. Lets iterate through iana first, then match against iapd.
	//If its not in iana having an iapd is useless.
	for routerID, route := range ianaPairs {
		//check if the routerID exists in iapdPairs
		_, exists := iapdPairs[routerID]
		if exists {
			//Add the route to the routeList
			routeList[iapdPairs[routerID]] = route
		}
	}

	createLogs(fmt.Sprintf("Current routes in /config/dhcpdv6.leases %s", routeList), true)
	return routeList
}

// Find the router ID and WAN address
func findIANA(entry string) (string, string) {
	//Find Identity Association for the router (IA-NA)
	patt := `}\s*(.*?)\s*{`
	re := regexp.MustCompile(patt)
	match := re.FindString(entry)
	if len(match) < 4 { //Catch out of bounds. Skip if the match is shorther then the lenghth 4, the minimum string size
		return "", ""
	}
	iana := match[1 : len(match)-3]

	//Find the routers WAN address. We will point routes to here. iaaddr
	patt2 := `iaaddr\s+([^\s{]+)\s*{`
	re2 := regexp.MustCompile(patt2)
	match2 := re2.FindString(entry)
	if len(match2) < 9 { //Catch out of bounds. Skip if the match is shorther then 9, the minimum string size
		return "", ""
	}
	iaaddr := match2[7 : len(match2)-2]

	return iana, iaaddr
}

// find the router ID and IPv6 subnet that was handed out.
func findIAPD(entry string) (string, string) {
	//Find Identity Association for the router (IA-PD)
	patt := `}\s*(.*?)\s*{`
	re := regexp.MustCompile(patt)
	match := re.FindString(entry)
	if len(match) < 4 { //Catch out of bounds. Skip if the match is shorther then the lenghth 4, the minimum string size
		return "", ""
	}
	iapd := match[1 : len(match)-3]

	//Find the IPv6 subnet routed to this router
	patt2 := `iaprefix\s+([^\s{]+)\s*{`
	re2 := regexp.MustCompile(patt2)
	match2 := re2.FindString(entry)
	if len(match2) < 11 { //Catch out of bounds. Skip if the match is shorther then 9, the minimum string size
		return "", ""
	}
	iaprefix := match2[9 : len(match2)-2]

	return iapd, iaprefix
}

// Get current ipv6 routes
func getCurrentRoutes() map[string]string {
	// Find the PD Agg subnets, we only want to make changes on these. Returns []*net.IPNet
	pdSlice := fidnPDAggSubnets()

	//Get the current routes. Save results in routesDirty.
	routeCfg := getConfig("run show configuration commands \"protocols static route6\"\n")

	//Parse the routes, return as key value pairs.
	currentRoutes := routeParser(routeCfg, pdSlice)

	createLogs(fmt.Sprintf("Current installed routes %s", currentRoutes), true)
	return currentRoutes
}

// Find the PD Agg subnets.
func fidnPDAggSubnets() []pdnet {
	//Variabels
	var pdSlice []pdnet

	//Get config for the pd server
	configs := getConfig("run show configuration commands \"service dhcpv6-server\"\n")

	//Parse through configs, return pdSlice
	for _, config := range configs {
		if strings.Contains(config, "prefix-delegation") { //Look for strings that contiant the subnet, start and stop
			//Find the subnet
			subStart := strings.Index(config, "subnet") + 8
			subEnd := strings.Index(config, "prefix-delegation") - 2
			subnetStr := config[subStart:subEnd]
			_, subnet, err := net.ParseCIDR(subnetStr)
			if err != nil {
				createLogs(fmt.Sprintf("fidnPDAggSubnets could not convert %s to *net.IP: %s", subnetStr, err), true)
				continue
			}

			//Find start
			startStart := strings.Index(config, "start") + 7
			startEnd := strings.Index(config, "stop") - 2
			startStr := config[startStart:startEnd]
			start, err := netip.ParseAddr(startStr)
			if err != nil {
				createLogs(fmt.Sprintf("fidnPDAggSubnets could not convert %s to netip.Addr: %s", startStr, err), true)
				continue
			}

			//Find stop
			stopStart := strings.Index(config, "stop") + 6
			stopEnd := strings.Index(config, "prefix-length") - 2
			stopStr := config[stopStart:stopEnd]
			stop, err := netip.ParseAddr(stopStr)
			if err != nil {
				createLogs(fmt.Sprintf("fidnPDAggSubnets could not convert %s to netip.Addr: %s", stopStr, err), true)
				continue
			}

			//Update pdSlice
			pdSlice = append(pdSlice, pdnet{Subnet: subnet, Start: start, Stop: stop})
		}
	}

	return pdSlice
}

// Parse through current routes, return as a map of rotues
func routeParser(routeCfg []string, pdSlice []pdnet) map[string]string {
	//Variabels
	currentRoutes := make(map[string]string) //Map of routes [subnet]destination

	//Iterate through the routeCfg, convert each route its own routeEntrie. Make sure config contains next-hop and is long enough.
	for _, config := range routeCfg {
		if strings.Contains(config, "next-hop") && len(config) > 14 {
			//Find the subnet
			subspos := 12
			subepos := strings.Index(config, "next-hop") - 2
			subnetStr := config[subspos:subepos]

			//Find the route
			rspos := strings.Index(config, "next-hop") + 10
			repos := len(config) - 1
			routeStr := config[rspos:repos]

			//convert routeStr to *net.IPNet
			_, subnet, err := net.ParseCIDR(subnetStr)
			if err != nil {
				continue
			}

			//Check if the current route is inside of the PD subnet.
			for _, pd := range pdSlice {
				if pd.Subnet.Contains(subnet.IP) {
					//Convert subnet.IP to netip.Addr
					net, err := netip.ParseAddr(subnet.IP.String())
					if err != nil {
						createLogs(fmt.Sprintf("routeParser could not convert %s to netip.Addr: %s", subnet.IP, err), true)
						continue
					}

					//Check if the subnet is within the subnet range. If so, add to currentRoutes
					if pd.Start.Compare(net) <= 0 && pd.Stop.Compare(net) >= 0 {
						currentRoutes[subnetStr] = routeStr
						createLogs(fmt.Sprintf("routeParser added route %s %s to currentRoutes:", subnetStr, routeStr), true)
					}
				}
			}
		}
	}

	return currentRoutes
}

// Compare routes and leases, get back list of routes to change (commands)
func compareRoutes(currentRoutes map[string]string, routeList map[string]string) ([]string, map[string]string) {
	//Variabels
	addRoutes := make(map[string]string)
	var removeRoutes []string

	//check if subnet exists in routeList, if it does check if the route is the same. If either fails, add the subnet to remove.
	for csubnet, croute := range currentRoutes {
		_, exists := routeList[csubnet]
		if exists { //If the subnet exists, check if the route is the same. If not, remove.
			if routeList[csubnet] != croute {
				removeRoutes = append(removeRoutes, csubnet)
			}
		} else { //Subnet does not exist in leases, remove
			removeRoutes = append(removeRoutes, csubnet)
		}
	}

	//check if there is an existing subnet for leases. If there is not, add it to the addRoutes list. Othwerwise ignore
	for subnet, route := range routeList {
		_, exists := currentRoutes[subnet]
		if exists { //If subnet exists, check if the route is the same. If not, add new route (old route will be removed by the above lines).
			if currentRoutes[subnet] != route {
				addRoutes[subnet] = route
			}
		} else {
			//The route does not exists, add to the new routes
			addRoutes[subnet] = route
		}
	}

	//Log results
	createLogs(fmt.Sprintf("Routes to remove: %s", removeRoutes), true)
	createLogs(fmt.Sprintf("Routes to add: %s", addRoutes), true)

	return removeRoutes, addRoutes
}

// Issue updates to edgerouter
func updateEdgerouter(removeRoutes []string, addRoutes map[string]string) {
	//Updates needed, set to false, unless new commands are passed. We dont want to update if there are no updates.
	updatesNeeded := false

	//Setup command script. I could not get this to work with the API, the script is just a wrapper around the API.
	//https://docs.vyos.io/en/latest/automation/command-scripting.html
	var commands string

	//Build commands for removeing routes
	for _, route := range removeRoutes {
		commands += fmt.Sprintf("delete protocols static route6 %s\n", route)
		updatesNeeded = true
	}

	//Build commands for installing new routes
	for subnet, route := range addRoutes {
		commands += fmt.Sprintf("set protocols static route6 %s next-hop %s\n", subnet, route)
		updatesNeeded = true
	}

	//If there are updates, push the new updates
	if updatesNeeded {
		setConfig(commands)
	}
}

// Log changes
func logChanges(removeRoutes []string, addRoutes map[string]string) {
	//Log removedRoutes
	for _, rm := range removeRoutes {
		createLogs(fmt.Sprintf("Removed route for %s", rm), false)
	}

	//Log added routes
	for subnet := range addRoutes {
		createLogs(fmt.Sprintf("Added route for %s", subnet), false)
	}
}
