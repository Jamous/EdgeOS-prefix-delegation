#!/bin/sh
#Build mips and mips64 binaries
GOARCH=mipsle GOOS=linux GOMIPS=softfloat go build -o "bin/mips/dhcpv6-pd" .
GOARCH=mips64 GOOS=linux GOMIPS=hardfloat go build -o "bin/mips64/dhcpv6-pd" .

#Reset build envireoment
unset GOOS unset GOARCH unset GOMIPS
