#!/bin/bash
#
# Set up sysctl for socket(AF_INET, SOCK_DGRAM, IPPROTO_ICMP)

sysctl net.ipv4.ping_group_range="0 0"

exec "$@"
