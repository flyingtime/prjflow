#!/bin/bash

export AUTOSSH_GATETIME=0

autossh -f -M 0 \
        -o "ServerAliveInterval=15" \
        -o "ServerAliveCountMax=3" \
        -o "ConnectTimeout=10" \
        -NT -R 0.0.0.0:8080:127.0.0.1:8080 \
        aiweb