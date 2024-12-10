EdgeOS-prefix-delegation
========================

This program is designed to install IPv6 PD routes into the routing table on Ubiquiti Edgerouters running EdgeOS. EdgeOS implements isc-dhcp-4.1 for both IPv4 and IPv6 DHCP functions. When enabled, the ISC server will pass out local IPv6 addresses via DHCPv6 to connected devices. It will also pass out IPv6 subnets via DHCPv6 Prefix Delegation, however routes for these leases are never installed into the routing table. This program reads the isc dhcp lease file, matches v6 subnets and destinations, and then installs them into the routing table. It will also prune any expired routes from the routing table.

Features
--------
* Currently running via cron once per minute
* Logs to system logs at /var/log/messages
* Supports debug logging with `-d`
* Checks to see if a user is logged in at run time. Can be bypassed with `-b`

Setup
-----
There is a setup script in this repo `setup.sh`. The command below will download the script and setup the program, including systemd units.
```
curl -O https://raw.githubusercontent.com/Jamous/EdgeOS-prefix-delegation/refs/heads/main/setup.sh
vbash setup.sh
```

Running
-------
This program is installed at `/bin/dhcpv6-pd`. The following Systemd services are created to run it
* `dhcpv6-pd.timer`: Launches the program at /bin/dhcpv6-pd
* `dhcpv6-pd.timer`: Triggers dhcpv6-pd.service every minute

You can check the status of these services with `systemctl status dhcpv6-pd.service` and `systemctl status dhcpv6-pd.timer`. Example output below.
```
admin@dhcpv6-pd:~$ systemctl status dhcpv6-pd.service
* dhcpv6-pd.service - EdgeOS prefix delegation. https://github.com/Jamous/EdgeOS-prefix-delegation
   Loaded: loaded (/etc/systemd/system/dhcpv6-pd.service; static; vendor preset: enabled)
   Active: inactive (dead) since Tue 2024-12-10 21:02:46 UTC; 20s ago
  Process: 4905 ExecStart=/bin/dhcpv6-pd -d -b (code=exited, status=0/SUCCESS)
 Main PID: 4905 (code=exited, status=0/SUCCESS)
admin@dhcpv6-pd:~$ systemctl status dhcpv6-pd.timer  
* dhcpv6-pd.timer - Run dhcpv6-pd.service Every Minute
   Loaded: loaded (/etc/systemd/system/dhcpv6-pd.timer; enabled; vendor preset: enabled)
   Active: active (waiting) since Tue 2024-12-10 20:21:44 UTC; 41min ago
```

Configuration
=============
For configuration you will need the following elements:
* An IPv6 address on the upstream port, in this example `eth0`.
* An IPv6 address on the interface that will be handing out PD subnets, in this example `switch0`.
* IPv6 router advertisements on the PD interface, in this example `switch0`.
* A static IPv6 default route. In this example assume that `2001:db8::2/64` on `eth0` is connected to `2001:db8::1` on an upstream router.

Below is the short version of what this config will look like. To see the full config look under config_example.

Network design
--------------
WAN address: 2001:db8::2/64<br>
LAN address: 2001:db8:1::1/64<br>
Prefix Delegation subnet: 2001:db8:1::/48 <br>
Prefix Delegation range: 2001:db8:1:1:: - 2001:db8:1:ffff:: <br>
Prefix Delegation size: /64<br>



Config Example
--------------
```
interfaces {
    ethernet eth0 {
        address dhcp
        address 2001:db8::2/64
    }
    switch switch0 {
        address 2001:db8:1::0/64
        ipv6 {
            dup-addr-detect-transmits 1
            router-advert {
                cur-hop-limit 64
                link-mtu 0
                managed-flag true
                max-interval 600
                other-config-flag false
                prefix 2001:db8:1::/64 {
                    autonomous-flag false
                    on-link-flag true
                    valid-lifetime 2592000
                }
                reachable-time 0
                retrans-timer 0
                send-advert true
            }
        }
    }
}
protocols {
    static {
        route6 ::/0 {
            next-hop 2001:db8::1 {
            }
        }
    }
}
service {
    dhcpv6-server {
        shared-network-name v6pd {
            subnet 2001:db8:1::/48 {
                address-range {
                    prefix 2001:db8:1::/64 {
                    }
                }
                prefix-delegation {
                    start 2001:db8:1:1:: {
                        stop 2001:db8:1:ffff:: {
                            prefix-length 64
                        }
                    }
                }
            }
        }
    }
}
```

Config commands example
-----------------------
```
set interfaces ethernet eth0 address dhcp
set interfaces ethernet eth0 address 2001:db8::2/64
set interfaces switch switch0 address 2001:db8:1::0/64
set interfaces switch switch0 ipv6 dup-addr-detect-transmits 1
set interfaces switch switch0 ipv6 router-advert cur-hop-limit 64
set interfaces switch switch0 ipv6 router-advert link-mtu 0
set interfaces switch switch0 ipv6 router-advert managed-flag true
set interfaces switch switch0 ipv6 router-advert max-interval 600
set interfaces switch switch0 ipv6 router-advert other-config-flag false
set interfaces switch switch0 ipv6 router-advert prefix 2001:db8:1::/64 autonomous-flag false
set interfaces switch switch0 ipv6 router-advert prefix 2001:db8:1::/64 on-link-flag true
set interfaces switch switch0 ipv6 router-advert prefix 2001:db8:1::/64 valid-lifetime 2592000
set interfaces switch switch0 ipv6 router-advert reachable-time 0
set interfaces switch switch0 ipv6 router-advert retrans-timer 0
set interfaces switch switch0 ipv6 router-advert send-advert true
set protocols static route6 ::/0 next-hop 2001:db8::1
set service dhcpv6-server shared-network-name v6pd subnet 2001:db8:1::/48 address-range prefix 2001:db8:1::/64
set service dhcpv6-server shared-network-name v6pd subnet 2001:db8:1::/48 prefix-delegation start 2001:db8:1:1:: stop 001:db8:1:ffff:: prefix-length 64
```

Code explanation
================
* Checks if a user is logged in before proceeding. Prevents config changes while in use.
* Reads isc dhcp logs at `/config/dhcpdv6.leases`.
    * Matches the client routers Identity Association for Non-temporary Addresses, (ia-na) with the routers duid. 
    * Matches the client routers Identity Association for Prefix Delegation, (ia-pd) based on duid with an assigned prefix.
    * Matches ia-na addresses with ia-pd subnets to determine new routes.
* Gets all subnets currently configured for prefix delegation.
* Reads all current IPv6 routes for subnets assigned for prefix delegation.
    * EdgeOS allows selecting a range within a subnet for prefix delegation, this program will only match subnets within that range.
* Generates a list of installed routes that have expired from the isc dhcp leases file to be removed.
* Generates a list of new routes to install.
* Removes unused routes using the Vyatta cli api.
* Installs new routes using the Vyatta cli api.
* Exits 

Exit codes
----------
* 0: Program completed successfully
* 1: Program encountered a fatal error, check log at /var/log/messages
* 2: User was logged in, program terminated

Debugging
=========
This program uses the system logging utility. Logs are stored at `/var/log/messages`. You can quickly see all related logs with `tail /var/log/messages | grep dhcpv6-pd`. This program has two command line switches
* `-b` Bypass check to see if a user is active on the router.
* `-d` Enable debug logs

You can manually run the program with debug logging and ignoring user logged in like this `/bin/dhcpv6-pd -b -d`.

To pass either of these commands at rutime, edit `/etc/systemd/system/dhcpv6-pd.service`. Ex.
```
[Unit]
Description=EdgeOS prefix delegation. https://github.com/Jamous/EdgeOS-prefix-delegation

[Service]
ExecStart=/bin/dhcpv6-pd -b -d
Type=oneshot
```

Then reload systemctl and restart the service. Dont forget, this will run until you revert the changes. Running setup.sh will overwrite any changes.

```
sudo systemctl daemon-reload
sudo systemctl restart dhcpv6-pd.service
```



VyOS Command Scripting
======================
All commands are issued via command scripting. We create a string of commands, then pass them as shell script to vbash.

This code sample uses VyOS Command Scripting to preform these actions:
* Add a route: 2001:db8:0:2::/64 via 2001:db8::5
* Remove a route: 2001:db8:0:1::/64
* Commit changes

```
#!/bin/vbash
source /opt/vyatta/etc/functions/script-template
configure
set protocols static route6 2001:db8:0:2::/64 next-hop 2001:db8::5
delete protocols static route6 2001:db8:0:1::/64
commit
exit
exit
```
VyOS API
==================
I can not get the API session to start from a script. I will leave this config in here for reference. The API is good for getting config elements from within a go program.

Get config
----------
This code sample will return all static IPv6 routes on the router. This can be run without having to enter a cli shell session.
```
/opt/vyatta/sbin/my_cli_shell_api showConfig protocols static route6
```

Set config
----------
This code sample uses the VyOS API to preform these actions:
* Add a route: 2001:db8:0:2::/64 via 2001:db8::5
* Remove a route: 2001:db8:0:1::/64
* Commit changes

```
session_env=$(/opt/vyatta/sbin/my_cli_shell_api getSessionEnv $PPID)
eval $session_env
cli-shell-api setupSession
/opt/vyatta/sbin/my_delete protocols static route6 2001:db8:0:1::/64
/opt/vyatta/sbin/my_set protocols static route6 2001:db8:0:2::/64 next-hop 2001:db8::5
/opt/vyatta/sbin/my_commit
cli-shell-api teardownSession
```

References
==========
* [VyOS CLI Shell API](https://vyos.dev/w/development/cli-shell-api/)
* [VyOS Command Scripting](https://docs.vyos.io/en/latest/automation/command-scripting.html)
* [VyOS Networks Blog: Versions mystery revealed](https://blog.vyos.io/versions-mystery-revealed)
