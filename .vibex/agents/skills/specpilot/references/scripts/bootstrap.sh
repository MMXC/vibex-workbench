#!/bin/bash
# bootstrap.sh — start SpecPilot services in workspace
set -e

WS_ROOT="${1:-$(pwd)}"
SPECPILOT_DIR="$WS_ROOT/.specpilot"
MF_DIR="$WS_ROOT/.specpilot-mf"
DC_PORT="${2:-7890}"
MF_PORT="${3:-5177}"

# Seed env vars for api_server.py
export SPECPILOT_DC_PORT="$DC_PORT"
export SPECPILOT_MF_PORT="$MF_PORT"

# Start DC API
python3 "$SPECPILOT_DIR/api_server.py" &
DC_PID=$!

# Start MF dev server
python3 -m http.server "$MF_PORT" --directory "$MF_DIR" &
MF_PID=$!

echo "DC_PID=$DC_PID MF_PID=$MF_PID"
echo "DC_URL=http://127.0.0.1:$DC_PORT"
echo "MF_URL=http://localhost:$MF_PORT"
