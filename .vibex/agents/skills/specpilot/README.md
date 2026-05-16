# SpecPilot Datalayer Skill

## Overview

This skill exposes the SpecPilot four-layer data capabilities to the VibeX Go agent.

## Files

- `SKILL.md` — Main skill file with CLI command reference and four-layer integration patterns
- `references/` — Additional reference docs
- `scripts/` — Helper scripts

## Usage

The Go agent loads this skill via `skill_load specpilot-datalayer`.
After loading, the agent can call SpecPilot CLI commands via `bash`.

## CLI Location

```
/tmp/specpilot/specpilot
```

Requires Python 3.7+.
