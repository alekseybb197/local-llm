#!/bin/bash

launchctl list | grep -q com.user.llm-server || \
  launchctl load ~/Library/LaunchAgents/com.user.llm-server.plist

launchctl list | grep com.user.llm-server

