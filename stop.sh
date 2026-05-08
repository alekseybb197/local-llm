#!/bin/bash

launchctl list | grep -q com.user.llm-server && \
  launchctl unload ~/Library/LaunchAgents/com.user.llm-server.plist


