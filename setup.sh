#!/bin/bash

#Get system architecture
arch=$(dpkg --print-architecture)
echo "$arch"

#Check if mips64
if [ "$arch" = "mips" ]; then
    #Install program and set permissions
    curl -O https://github.com/Jamous/EdgeOS-prefix-delegation/raw/refs/heads/main/bin/mips64/dhcpv6-pd
    chmod 777 dhcpv6-pd
    
    #Run program
    ./dhcpv6-pd -d -b 
    tail /var/log/messages
elif [ "$arch" = "mipsel" ]; then
    #Install program and set permissions
    curl -O https://github.com/Jamous/EdgeOS-prefix-delegation/raw/refs/heads/main/bin/mips/dhcpv6-pd
    chmod 777 dhcpv6-pd
    
    #Run program
    ./dhcpv6-pd -d -b 
    tail /var/log/messages
fi

