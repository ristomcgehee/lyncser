#!/bin/bash

# This script installs lyncser as a launchd job.

set -e

DIR_SCRIPT=$(dirname -- "${BASH_SOURCE[0]}")

sudo cp $DIR_SCRIPT/../lyncser /usr/local/bin
sudo touch /var/log/lyncser.log /var/log/lyncser-error.log
sudo chmod 666 /var/log/lyncser.log /var/log/lyncser-error.log

cp $DIR_SCRIPT/lyncser.plist ~/Library/LaunchAgents/lyncser.plist

launchctl load ~/Library/LaunchAgents/lyncser.plist
