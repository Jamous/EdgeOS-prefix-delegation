firewall {
    all-ping enable
    broadcast-ping disable
    group {
        network-group lan {
            description ""
            network 192.168.50.0/23
        }
    }
    ipv6-receive-redirects disable
    ipv6-src-route disable
    ip-src-route disable
    log-martians enable
    receive-redirects disable
    send-redirects enable
    source-validation disable
    syn-cookies enable
}
interfaces {
    ethernet eth0 {
        address dhcp
        address 2001:db8::2/64
        duplex auto
        speed auto
    }
    ethernet eth1 {
        duplex auto
        speed auto
    }
    ethernet eth2 {
        duplex auto
        speed auto
    }
    ethernet eth3 {
        duplex auto
        speed auto
    }
    ethernet eth4 {
        duplex auto
        poe {
            output off
        }
        speed auto
    }
    loopback lo {
    }
    switch switch0 {
        address 192.168.50.1/23
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
        mtu 1500
        switch-port {
            interface eth1 {
            }
            interface eth2 {
            }
            interface eth3 {
            }
            interface eth4 {
            }
            vlan-aware disable
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
    dhcp-server {
        disabled false
        hostfile-update disable
        shared-network-name dhcp_server {
            authoritative disable
            subnet 192.168.50.0/23 {
                default-router 192.168.50.1
                dns-server 8.8.8.8
                lease 86400
                start 192.168.50.10 {
                    stop 192.168.51.254
                }
            }
        }
        static-arp disable
        use-dnsmasq disable
    }
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
    gui {
        http-port 80
        https-port 443
        older-ciphers enable
    }
    nat {
        rule 5001 {
            description nat_all
            log disable
            outbound-interface eth0
            type masquerade
        }
    }
    ssh {
        port 22
        protocol-version v2
    }
}
system {
    analytics-handler {
        send-analytics-report false
    }
    crash-handler {
        send-crash-report false
    }
    host-name dhcpv6-pd
    login {
        user admin {
            authentication {
                encrypted-password $5$89rtgeo15UzHntOe$b4Gz1Cin6uv2hdrXHS9Tlj3Ly4jqO6QBPddjj4IyTq5
                plaintext-password ""
                public-keys key {
                    key AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBAwz+pRHPq2Ro+XzTfOu43YtuNxU3RDLWICU+LEBynr5BMGH1tt/ti1vOpZg1eNfotZMwoWPjWmnu8H98FkYbqA=
                    type ecdsa-sha2-nistp256
                }
            }
            level admin
        }
    }
    name-server 8.8.8.8
    ntp {
        server 0.ubnt.pool.ntp.org {
        }
        server 1.ubnt.pool.ntp.org {
        }
        server 2.ubnt.pool.ntp.org {
        }
        server 3.ubnt.pool.ntp.org {
        }
    }
    offload {
        hwnat enable
    }
    syslog {
        global {
            facility all {
                level notice
            }
            facility protocols {
                level debug
            }
        }
    }
    time-zone UTC
}


/* Warning: Do not remove the following line. */
/* === vyatta-config-version: "config-management@1:conntrack@1:cron@1:dhcp-relay@1:dhcp-server@4:firewall@5:ipsec@5:nat@3:qos@1:quagga@2:suspend@1:system@5:ubnt-l2tp@1:ubnt-pptp@1:ubnt-udapi-server@1:ubnt-unms@2:ubnt-util@1:vrrp@1:vyatta-netflow@1:webgui@1:webproxy@1:zone-policy@1" === */
/* Release version: v2.0.9-hotfix.6.5574651.221230.1015 */
