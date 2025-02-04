#!/bin/vbash
#Get system architecture
arch=$(dpkg --print-architecture)

#Download program for archiceture
echo "Downloading program"
if [ "$arch" = "mips" ]; then
    sudo curl -Lok /bin/dhcpv6-pd https://raw.githubusercontent.com/Jamous/EdgeOS-prefix-delegation/main/bin/mips64/dhcpv6-pd
elif [ "$arch" = "mipsel" ]; then
    sudo curl -Lok /bin/dhcpv6-pd https://raw.githubusercontent.com/Jamous/EdgeOS-prefix-delegation/main/bin/mips/dhcpv6-pd
else
    echo "Unsupported architecture: $arch"
    exit 1
fi

#Set program permissions
echo "Setting program permissions"
sudo chmod 755 /bin/dhcpv6-pd || { echo "Failed to set permissions"; exit 1; }

#Define and install systemd service
unit="[Unit]
Description=EdgeOS prefix delegation. https://github.com/Jamous/EdgeOS-prefix-delegation

[Service]
ExecStart=/bin/dhcpv6-pd
Type=oneshot"

echo "Installing dhcpv6-pd.service"
echo "$unit" | sudo tee /etc/systemd/system/dhcpv6-pd.service > /dev/null || { echo "Failed to create service file"; exit 1; }


#Define and install systemd timer
timer="[Unit]
Description=Run dhcpv6-pd.service Every Minute

[Timer]
OnCalendar=*:0/1
Unit=dhcpv6-pd.service

[Install]
WantedBy=timers.target"

echo "Installing dhcpv6-pd.timer"
echo "$timer" | sudo tee /etc/systemd/system/dhcpv6-pd.timer > /dev/null || { echo "Failed to create timer file"; exit 1; }

#Reload systemd to recognize new units
echo "Reloading systemd daemon..."
sudo systemctl daemon-reload || { echo "Systemd reload failed"; exit 1; }

#Start and enable the timer
echo "Starting and enabling dhcpv6-pd.timer"
sudo systemctl enable --now dhcpv6-pd.timer | { echo "Failed to enable and start timer"; exit 1; }

systemctl status dhcpv6-pd.timer

echo "Install complete. Sometimes the dhcpv6 will not start correctly. Restart the router to solve this."
