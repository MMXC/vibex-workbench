#!/bin/bash
# install.sh — install SpecPilot from skill references to .specpilot/ workspace dir
set -e

WS_ROOT="${1:-$(pwd)}"
SKILL_REF="$WS_ROOT/.vibex/agents/skills/specpilot/references"
INSTALL_DIR="$WS_ROOT/.specpilot"
MF_DIR="$WS_ROOT/.specpilot-mf"

mkdir -p "$INSTALL_DIR" "$MF_DIR"

# Copy CLI
cp -r "$SKILL_REF/cli/"* "$INSTALL_DIR/"

# Copy MF
cp "$SKILL_REF/mf/index.html" "$MF_DIR/"

echo "Installed SpecPilot to $INSTALL_DIR and $MF_DIR"
