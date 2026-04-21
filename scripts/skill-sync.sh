#!/usr/bin/env bash
# skill-sync.sh — 双向同步 live skills ↔ repo snapshots
#
# 用法：
#   ./scripts/skill-sync.sh push [skill-name]   # live → repo（单个或全部 tracked）
#   ./scripts/skill-sync.sh pull [skill-name]   # repo → live（恢复）
#   ./scripts/skill-sync.sh status              # 查看哪些 skill 有差异
#
# 环境变量：
#   SKILLS_LIVE_DIR   默认 /root/.hermes/skills
#   SKILLS_REPO_DIR   默认 $(pwd)/skills

set -euo pipefail

SKILLS_LIVE_DIR="${SKILLS_LIVE_DIR:-/root/.hermes/skills}"
SKILLS_REPO_DIR="${SKILLS_REPO_DIR:-$(cd "$(dirname "$0")/.." && pwd)/skills}"
MARKER="repo_tracked:"

# ── helpers ──────────────────────────────────────────────────────────────────

need() { command -v "$1" >/dev/null || { echo "need: $1" >&2; exit 1; }; }

# 读取 skill frontmatter 中的 repo_tracked 值
is_tracked() {
  local skill_dir="$1"
  local frontmatter="$skill_dir/SKILL.md"
  if [[ -f "$frontmatter" ]] && grep -q "^${MARKER}" "$frontmatter"; then
    grep "^${MARKER}" "$frontmatter" | awk '{print $2}' | tr -d ' '
  else
    echo ""
  fi
}

# 列出所有 tracked skills（处理两种目录结构）
# 结构A: category/skill-name/SKILL.md  (多 skill category)
# 结构B: single-skill-category/SKILL.md (单 skill category)
list_tracked() {
  local live="$SKILLS_LIVE_DIR"
  for cat_dir in "$live"/*/; do
    [[ -d "$cat_dir" ]] || continue
    cat_name=$(basename "$cat_dir")
    # 结构B: 直接在 category 目录下的 SKILL.md
    if [[ -f "$cat_dir/SKILL.md" ]]; then
      tracked=$(is_tracked "$cat_dir")
      if [[ "$tracked" == "true" ]]; then
        echo "$cat_name"
      fi
    fi
    # 结构A: category/skill/SKILL.md
    for skill_dir in "$cat_dir"*/; do
      [[ -d "$skill_dir" ]] || continue
      tracked=$(is_tracked "$skill_dir")
      if [[ "$tracked" == "true" ]]; then
        cat_name=$(basename "$cat_dir")
        skill_name=$(basename "$skill_dir")
        rel="${cat_name}/${skill_name}"
        echo "$rel"
      fi
    done
  done
}

# Compare content of a skill (recursively by file count + SKILL.md hash)
skill_differs() {
  local live="$1"
  local repo="$2"
  # Quick check: same number of files?
  live_count=$(find "$live" -type f 2>/dev/null | wc -l)
  repo_count=$(find "$repo" -type f 2>/dev/null | wc -l)
  if [[ "$live_count" != "$repo_count" ]]; then
    return 0  # differs
  fi
  # Compare SKILL.md hash (proxy for whole skill content)
  live_hash=$(md5sum "$live/SKILL.md" 2>/dev/null | awk '{print $1}')
  repo_hash=$(md5sum "$repo/SKILL.md" 2>/dev/null | awk '{print $1}')
  if [[ "$live_hash" != "$repo_hash" ]]; then
    return 0  # differs
  fi
  return 1  # same
}

# Compare single-file skill (direct SKILL.md)
single_differs() {
  local live="$1"
  local repo="$2"
  live_hash=$(md5sum "$live/SKILL.md" 2>/dev/null | awk '{print $1}')
  repo_hash=$(md5sum "$repo/SKILL.md" 2>/dev/null | awk '{print $1}')
  [[ "$live_hash" != "$repo_hash" ]]
}

# Compare timestamps (fallback)
live_newer_than_repo() {
  local live="$1"
  local repo="$2"
  live_ts=$(stat -c %Y "$live/SKILL.md" 2>/dev/null || stat -f %m "$live/SKILL.md" 2>/dev/null)
  repo_ts=$(stat -c %Y "$repo/SKILL.md" 2>/dev/null || stat -f %m "$repo/SKILL.md" 2>/dev/null)
  [[ "$live_ts" -gt "$repo_ts" ]]
}

# Compare timestamps (fallback)
repo_newer_than_live() {
  local live="$1"
  local repo="$2"
  live_ts=$(stat -c %Y "$live/SKILL.md" 2>/dev/null || stat -f %m "$live/SKILL.md" 2>/dev/null)
  repo_ts=$(stat -c %Y "$repo/SKILL.md" 2>/dev/null || stat -f %m "$repo/SKILL.md" 2>/dev/null)
  [[ "$repo_ts" -gt "$live_ts" ]]
}

# Determine status of a skill using hash comparison (authoritative) + ts (fallback)
skill_status() {
  local rel="$1"  # e.g. devops/vibex-qa-entry-points or vibex-agent-ops
  local live="$SKILLS_LIVE_DIR/$rel"
  local repo="$SKILLS_REPO_DIR/$rel"

  if [[ ! -d "$live" ]] && [[ ! -d "$repo" ]]; then
    echo "missing"
  elif [[ ! -d "$live" ]] && [[ -d "$repo" ]]; then
    echo "only-repo"
  elif [[ -d "$live" ]] && [[ ! -d "$repo" ]]; then
    echo "only-live"
  else
    # Both exist — use hash comparison (authoritative)
    if [[ -f "$live/SKILL.md" ]] && [[ -d "$repo" ]] && [[ ! -f "$repo/SKILL.md" ]]; then
      # Multi-file skill (repo is missing) → differs
      echo "differs"
    elif single_differs "$live" "$repo"; then
      # Single-file or multi-file skill with different content
      if live_newer_than_repo "$live" "$repo"; then
        echo "live-newer"
      elif repo_newer_than_live "$live" "$repo"; then
        echo "repo-newer"
      else
        # Same content but ts differs (e.g. after push) — use content hash
        echo "differs"
      fi
    else
      echo "synced"
    fi
  fi
}

# ── commands ────────────────────────────────────────────────────────────────

cmd_status() {
  echo "=== Skill Sync Status ==="
  echo "Live : $SKILLS_LIVE_DIR"
  echo "Repo : $SKILLS_REPO_DIR"
  echo ""
  echo "Tracked skills:"
  tracked_count=0
  while IFS= read -r rel; do
    [[ -z "$rel" ]] && continue
    status=$(skill_status "$rel")
    case "$status" in
      synced)       icon="✅" ;;
      live-newer)   icon="🔼" ;;
      repo-newer)   icon="🔽" ;;
      only-live)    icon="📦" ;;
      only-repo)    icon="📥" ;;
      missing)      icon="❌" ;;
    esac
    printf "  %s [%s] %s\n" "$icon" "$status" "$rel"
    ((tracked_count++)) || true
  done < <(list_tracked)
  echo ""
  echo "Total: $tracked_count tracked skills"
}

cmd_push() {
  local target="${1:-}"
  echo "=== Pushing tracked skills: live → repo ==="

  if [[ -n "$target" ]]; then
    skills_to_push=("$target")
  else
    while IFS= read -r rel; do
      [[ -z "$rel" ]] && continue
      skills_to_push+=("$rel")
    done < <(list_tracked)
  fi

  pushed=0
  for rel in "${skills_to_push[@]:-}"; do
    status=$(skill_status "$rel")
    if [[ "$status" == "repo-newer" ]]; then
      echo "  ⏭  SKIP (repo newer): $rel — manual merge needed"
      continue
    fi
    if [[ "$status" == "synced" ]]; then
      echo "  ✓  $rel (unchanged)"
      continue
    fi
    if [[ "$status" == "only-repo" ]]; then
      echo "  ⏭  SKIP (only in repo): $rel"
      continue
    fi

    src="$SKILLS_LIVE_DIR/$rel"
    dst="$SKILLS_REPO_DIR/$rel"
    if [[ ! -d "$src" ]]; then
      echo "  ❌ NOT FOUND: $rel"
      continue
    fi

    mkdir -p "$(dirname "$dst")"
    rm -rf "$dst"
    cp -r "$src" "$dst"
    echo "  ✅ $rel"
    ((pushed++)) || true
  done
  echo "Pushed $pushed skill(s)"
}

cmd_pull() {
  local target="${1:-}"
  echo "=== Pulling tracked skills: repo → live ==="

  if [[ -n "$target" ]]; then
    skills_to_pull=("$target")
  else
    while IFS= read -r rel; do
      [[ -z "$rel" ]] && continue
      skills_to_pull+=("$rel")
    done < <(list_tracked)
  fi

  pulled=0
  for rel in "${skills_to_pull[@]:-}"; do
    status=$(skill_status "$rel")
    if [[ "$status" == "only-live" ]]; then
      echo "  ⏭  SKIP (only in live, not tracked in repo): $rel"
      continue
    fi
    if [[ "$status" == "repo-newer" ]] || [[ "$status" == "only-repo" ]]; then
      src="$SKILLS_REPO_DIR/$rel"
      dst="$SKILLS_LIVE_DIR/$rel"
      if [[ ! -d "$src" ]]; then
        echo "  ❌ NOT FOUND in repo: $rel"
        continue
      fi
      mkdir -p "$(dirname "$dst")"
      rm -rf "$dst"
      cp -r "$src" "$dst"
      echo "  ✅ $rel"
      ((pulled++)) || true
    elif [[ "$status" == "live-newer" ]]; then
      echo "  ⚠  WARNING (live newer than repo): $rel — not overwriting live"
    else
      echo "  ✓  $rel (unchanged)"
    fi
  done
  echo "Pulled $pulled skill(s)"
}

# ── main ────────────────────────────────────────────────────────────────────

cmd="${1:-}"
shift || true

case "$cmd" in
  push)   cmd_push "$@" ;;
  pull)   cmd_pull "$@" ;;
  status) cmd_status ;;
  *) cat <<EOF
Usage: $0 <command> [skill-name]

Commands:
  push [skill]   Push tracked skills: /root/.hermes/skills → repo/skills
                 (Omit skill name to push ALL tracked)
  pull [skill]   Pull tracked skills: repo/skills → /root/.hermes/skills
                 (Omit skill name to pull ALL tracked)
  status         Show sync status of all tracked skills

Environment:
  SKILLS_LIVE_DIR   default: /root/.hermes/skills
  SKILLS_REPO_DIR   default: \$PWD/skills

Tracked skills are identified by \`repo_tracked: true\` in their SKILL.md frontmatter.
EOF
    ;;
esac
