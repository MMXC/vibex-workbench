#!/usr/bin/env bash
# 将本机 Cursor「Team Kit」插件内的 skills 复制到仓库 skills/，供无 Cursor 环境（CI、远程 agent）使用。
# 仓库根目录执行:
#   bash scripts/sync-cursor-team-kit-skills.sh
# 指定插件内 skills 路径（可选）:
#   TEAM_KIT_SKILLS_SRC=/path/to/cursor-team-kit/<hash>/skills bash scripts/sync-cursor-team-kit-skills.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="${ROOT}/skills"

find_default_src() {
  local home="${HOME:-}"
  [[ -n "$home" ]] || return 1
  shopt -s nullglob
  local candidates=("${home}/.cursor/plugins/cache/cursor-public/cursor-team-kit"/*/skills)
  shopt -u nullglob
  local skills
  for skills in "${candidates[@]}"; do
    [[ -d "$skills" ]] || continue
    [[ -f "${skills}/check-compiler-errors/SKILL.md" ]] || continue
    printf '%s' "$skills"
    return 0
  done
  return 1
}

SRC="${TEAM_KIT_SKILLS_SRC:-}"
if [[ -z "${SRC}" ]] || [[ ! -d "${SRC}" ]]; then
  SRC="$(find_default_src)" || true
fi

if [[ -z "${SRC}" ]] || [[ ! -d "${SRC}" ]]; then
  echo "sync-cursor-team-kit-skills: 找不到 Team Kit skills 目录。" >&2
  echo "请在本机安装 Cursor Team Kit，或设置 TEAM_KIT_SKILLS_SRC 指向 …/cursor-team-kit/<hash>/skills" >&2
  exit 1
fi

echo "源: ${SRC}"
echo "目标: ${DEST}"

count=0
for d in "${SRC}"/*/; do
  [[ -d "$d" ]] || continue
  name="$(basename "$d")"
  [[ -f "${d}/SKILL.md" ]] || continue
  mkdir -p "${DEST}/${name}"
  cp -f "${d}/SKILL.md" "${DEST}/${name}/SKILL.md"
  count=$((count + 1))
done

echo "已复制 ${count} 个 SKILL.md 到 skills/<名称>/"
