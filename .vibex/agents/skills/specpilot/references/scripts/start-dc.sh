#!/bin/bash
# Start SpecPilot DataCenter as true daemon (survives parent death)
# Usage: start-dc.sh <port> <workspace>
set -e

PORT="${1:-7890}"
WORKSPACE="${2:-.}"

cd "$WORKSPACE"
exec python3 -m specpilot.dc --port "$PORT"
