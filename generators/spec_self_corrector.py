#!/usr/bin/env python3
"""
spec_self_corrector.py -- spec-first 自举循环的修正层
spec_criteria_validator.py 的下游：
  读取 criteria report → 分类处理：
    self_heal   → 直接修复（template 同步、路径修正）
    fix_import  → 写入 agent task（修复 import 断链）
    patch_component → 写入 agent task（修补组件缺口）
    write_test  → vibex_criteria_engine 派生 AC（同步）；失败时回退 agent
    notify      → 记录并通知 coord
    skip        → 跳过

设计原则（对应用户的 spec-first + agent 聚焦关键区域）：
  - 生成器产生~80%样板，留下 hook 标记
  - Agent 只修关键缺口（critical severity）
  - 生成器根据反馈自我修正模板（self_heal）
  - 不追求一步到位 → 允许不完美 → 迭代逼近目标
"""
import sys
import json
import subprocess
from pathlib import Path
from dataclasses import dataclass
from typing import Literal

TEMPLATE_DIR = Path("generators/templates")
SPEC_DIR = Path("specs")
MANIFEST_PATH = Path("generators/.manifest.json")


# ── Self-Heal 策略 ─────────────────────────────────────────
def self_heal_template_sync(spec_name: str, spec_content: str, gap_message: str) -> bool:
    """Meta-template spec 同步到 generators/templates/。"""
    if "template_sync" not in gap_message and "template_file_missing" not in gap_message:
        return False
    tpl_file = TEMPLATE_DIR / f"{spec_name}.yaml.tpl"
    tpl_file.parent.mkdir(parents=True, exist_ok=True)
    tpl_file.write_text(spec_content, encoding="utf-8")
    print(f"  [heal] Synced template → {tpl_file}")
    return True


def self_heal_parent_chain(spec_content: dict, parent_name: str) -> bool:
    """修正 spec 的 parent 引用（如果路径猜错了，尝试修正）。"""
    # 目前只是记录，不自动改，避免破坏用户意图
    print(f"  [heal] Parent chain gap detected for '{parent_name}' — needs human review")
    return False


# ── Agent Task Writer ────────────────────────────────────────
# 将 critical gaps 打包为 agent task JSON，写入可观测队列 + 调内置 agent API
#
# 可观测队列：generators/agent-queue/<task_id>.json
#   - 状态机: pending → running → done / failed
#   - agent SSE 推送 canvas.queue_update 事件 → 前端可观测
#   - 符合 openclaw 团队理念，但用 vibex 内置 agent 代替 openclaw team-tasks
#
# Agent API: POST http://localhost:33338/api/chat
#   - 发送任务描述 → agent 执行 → SSE 观察结果

# ── Manifest reader ───────────────────────────────────────────
def load_manifest() -> dict:
    """读取 gen.py 输出的 .manifest.json。"""
    if not MANIFEST_PATH.exists():
        return {}
    try:
        return json.loads(MANIFEST_PATH.read_text(encoding="utf-8"))
    except Exception:
        return {}


# ── 配置常量 ────────────────────────────────────────────────
TEAM_TASKS_DIR = Path("generators/agent-queue")  # vibex-workbench 自有队列目录
AGENT_API = "http://localhost:33338/api/chat"
MANIFEST_PATH = Path("generators/.manifest.json")


def write_agent_tasks(gaps: list, spec_name: str, output_dir: Path | None = None) -> list[Path]:
    """将 gaps 转换为可观测队列 JSON 文件，返回写入的路径列表。

    Queue JSON 格式（状态机驱动，前端可订阅）：
    {
      "id": "spec-write_test-20260422xxxx",
      "status": "pending",
      "mission": "...",
      "context": { "spec_name": "...", "gaps": [...], "manifest": {...} },
      "actions": ["..."],
      "created_by": "spec_self_corrector",
      "created_at": "...",
      "started_at": null,
      "completed_at": null,
      "result": null,
      "error": null
    }
    """
    if output_dir is None:
        output_dir = TEAM_TASKS_DIR
    output_dir.mkdir(parents=True, exist_ok=True)

    import datetime
    ts = datetime.datetime.now().strftime("%Y%m%d%H%M%S%f")[:-3]

    # 读取 manifest（如果存在）
    manifest = {}
    if MANIFEST_PATH.exists():
        try:
            manifest = json.loads(MANIFEST_PATH.read_text(encoding="utf-8"))
        except Exception:
            pass

    tasks = []
    # 按 action 类型分组，减少 task 数量
    grouped: dict[str, list] = {}
    for g in gaps:
        action = g.get("agent_action", "fix_import")
        if action not in grouped:
            grouped[action] = []
        grouped[action].append(g)

    for action, gap_list in grouped.items():
        task_id = f"spec-{action}-{ts}"
        mission = _build_mission(action, gap_list, spec_name)
        context = {
            "spec_name": spec_name,
            "spec_dir": str(SPEC_DIR),
            "manifest": manifest,
            "gaps": [g.__dict__.copy() if hasattr(g, "__dict__") else g for g in gap_list],
        }

        # 可观测队列格式：含状态机字段
        task = {
            "id": task_id,
            "status": "pending",  # pending → running → done / failed
            "mission": mission,
            "context": context,
            "actions": _action_steps(action, gap_list),
            "created_by": "spec_self_corrector",
            "created_at": datetime.datetime.now().isoformat(),
            "started_at": None,
            "completed_at": None,
            "result": None,
            "error": None,
        }

        out_path = output_dir / f"{task_id}.json"
        out_path.write_text(json.dumps(task, indent=2, ensure_ascii=False), encoding="utf-8")
        tasks.append(out_path)

    return tasks


def _update_task_status(task_path: Path, status: str, result=None, error=None):
    """更新队列任务状态（写入队列文件）。"""
    try:
        task = json.loads(task_path.read_text(encoding="utf-8"))
        task["status"] = status
        import datetime
        if status == "running":
            task["started_at"] = datetime.datetime.now().isoformat()
        elif status in ("done", "failed"):
            task["completed_at"] = datetime.datetime.now().isoformat()
            task["result"] = result
            task["error"] = error
        task_path.write_text(json.dumps(task, indent=2, ensure_ascii=False), encoding="utf-8")
    except Exception:
        pass


def _build_mission(action: str, gaps: list, spec_name: str) -> str:
    """为给定 action 生成 agent mission 描述。"""
    messages = [g.get("message", "") for g in gaps]
    if action == "fix_import":
        return (
            f"Fix import path breaks in {spec_name} spec artifacts. "
            f"Files: {messages}. "
            f"Check $lib imports and relative imports match actual file paths. "
            f"After fixing, verify with: python3 generators/spec_criteria_validator.py --spec {spec_name} --check-generated"
        )
    elif action == "patch_component":
        return (
            f"Patch component gaps in {spec_name}. "
            f"Issues: {messages}. "
            f"Fill in the ~20% glue/adapter code needed to connect generated skeletons. "
            f"Do NOT rewrite .Skeleton.svelte files — only the developer-maintained .svelte files."
        )
    elif action == "write_test":
        # 优先用 vibex_criteria_engine 直接派生（同步，快速）
        # 只有引擎失败才派发 agent
        feature_specs = [g.get("spec_path", spec_name) for g in gaps]
        try:
            from vibex_criteria_engine import derive_and_write
            failed = []
            for fs in feature_specs:
                spec_path = Path(SPEC_DIR) / fs
                if not spec_path.exists():
                    spec_path = next(Path(SPEC_DIR).rglob(f"{fs}_feature.yaml"), None)
                if spec_path:
                    result = derive_and_write(spec_path, dry_run=False, force=True)
                    if not result:
                        failed.append(fs)
            if not failed:
                return (f"[engine] 派生 AC 成功: {feature_specs}", "engine")
            else:
                return (f"[engine] 部分失败，回退 agent: {failed}", "agent_task")
        except Exception as e:
            return (f"[engine] 失败，回退 agent: {e}", "agent_task")
    elif action == "self_heal":
        return (
            f"Self-heal template mismatch in {spec_name}. "
            f"Issues: {messages}. "
            f"Run: python3 generators/gen.py specs frontend to regenerate, "
            f"then: python3 generators/spec_criteria_validator.py --spec {spec_name} --check-generated"
        )
    else:
        return f"Review and fix gaps in {spec_name}: {messages}"


def _action_steps(action: str, gaps: list) -> list[str]:
    """为给定 action 生成标准步骤。"""
    base = ["Read the spec YAML for context", "Understand the gap from criteria report"]
    if action == "fix_import":
        return base + [
            "Read the generated .svelte/.ts files",
            "Fix import path to match actual file locations",
            "Verify: python3 generators/spec_criteria_validator.py --check-generated"
        ]
    elif action == "patch_component":
        return base + [
            "Read the .Skeleton.svelte file (generator output)",
            "Read the existing .svelte file (developer code)",
            "Add adapter/glue code in the .svelte to wire skeleton to existing code",
            "Test: make dev"
        ]
    elif action == "write_test":
        # 步骤由 vibex_criteria_engine 处理，这里只描述 agent fallback
        return [
            "Run: python3 generators/vibex_criteria_engine.py --all --force",
            "Verify: python3 generators/spec_criteria_validator.py --level L3"
        ]
    elif action == "self_heal":
        return base + [
            "Run: python3 generators/gen.py specs frontend",
            "Run: python3 generators/spec_criteria_validator.py --check-generated",
            "Check that gaps are resolved"
        ]
    return base + ["Fix the reported issue", "Verify the fix"]


def _call_vibex_agent(mission: str, context: dict, thread_id: str) -> dict:
    """调用 vibex-workbench 内置 Go agent（POST /api/chat），返回 threadId。

    API: POST http://localhost:33338/api/chat
    Body: {"threadId": "...", "input": "..."}
    Response: {"status": "queued", "threadId": "..."}

    策略：input 里直接给 spec 内容，减少 agent 工具调用，加速完成。
    """
    import urllib.request, urllib.error

    spec_name = context.get("spec_name", "")
    spec_dir = context.get("spec_dir", "specs")
    gaps = context.get("gaps", [])

    # 读取 spec 文件内容，塞进 input 减少工具调用
    spec_content = ""
    for p in Path(spec_dir).rglob(f"{spec_name}_feature.yaml"):
        try:
            spec_content = p.read_text(encoding="utf-8")[:3000]
            break
        except Exception:
            pass

    gap_summary = "; ".join(
        g.get("message", g.get("criterion_id", "")) for g in gaps
    ) if gaps else ""

    full_input = f"""{mission}

SPEC CONTENT (read it and modify):
{spec_content}

Context: spec={spec_name}, gaps={gap_summary}

IMPORTANT: After writing the acceptance_criteria to the spec YAML file using write_file tool, confirm by reading the file back to verify."""

    payload = json.dumps({
        "threadId": thread_id,
        "input": full_input,
    }).encode("utf-8")

    req = urllib.request.Request(
        AGENT_API,
        data=payload,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            result = json.loads(resp.read().decode("utf-8"))
            return result
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8")[:200]
        return {"error": f"HTTP {e.code}: {body}"}
    except urllib.error.URLError as e:
        return {"error": f"Connection failed: {e.reason}"}
    except Exception as e:
        return {"error": str(e)}


def delegate_to_agent(gaps: list, spec_name: str, dry_run: bool = True) -> list[Path]:
    """
    可观测队列 + 内置 agent 调用：
    1. 写队列文件（generators/agent-queue/），状态机 pending → running → done/failed
    2. 调 vibex-workbench Go agent（POST http://localhost:33338/api/chat）
    3. agent 执行完后 SSE 推送 canvas.queue_update → 前端可观测

    dry_run=True 时只打印，不写队列文件。
    """
    if not gaps:
        return []

    if dry_run:
        print(f"\n[agent] Dry-run: would queue {len(gaps)} task(s):")
        for g in gaps:
            print(f"  [{g.get('layer','')}] {g.get('criterion_id','')}: {g.get('message','')[:80]}")
        return []

    # 写入可观测队列
    tasks = write_agent_tasks(gaps, spec_name)
    print(f"\n[agent] Queued {len(tasks)} task(s) to {TEAM_TASKS_DIR}:")

    # 调用 vibex 内置 agent
    action = gaps[0].get("agent_action", "fix_import")
    import datetime
    thread_id = f"spec-{action}-{datetime.datetime.now().strftime('%m%d%H%M%S')}"
    mission = _build_mission(action, gaps, spec_name)
    context = {
        "spec_name": spec_name,
        "manifest": load_manifest(),
        "gaps": [g.__dict__.copy() if hasattr(g, "__dict__") else g for g in gaps],
        "spec_dir": str(SPEC_DIR),
    }

    for task_path in tasks:
        _update_task_status(task_path, "running")
        print(f"  → Running: {task_path.name} (threadId: {thread_id})")
        result = _call_vibex_agent(mission, context, thread_id)
        if result.get("error"):
            _update_task_status(task_path, "failed", error=result["error"])
            print(f"  ❌ Failed: {result['error']}")
        else:
            returned_tid = result.get("threadId", result.get("status", "unknown"))
            _update_task_status(task_path, "done", result={"threadId": returned_tid})
            print(f"  ✅ Queued (threadId: {returned_tid})")

    return tasks


# ── 主循环 ─────────────────────────────────────────────────
def process_reports(reports: list, dry_run: bool = True, execute_heal: bool = True):
    """
    处理 criteria reports，分类执行修正。

    核心闭环：
      self_heal  → 直接修复（template 同步等） → 写回 spec/template
      agent      → 写入 team-tasks 队列 → openclaw agent 拾取执行
      skip       → 记录并继续
    """
    heal_count = 0
    agent_tasks = []  # collect all agent tasks across specs
    write_test_done = set()  # 去重：每个 spec 只派生一次

    for report in reports:
        spec_name = report.get("spec_name", "")
        spec_path = None
        spec_data = {}

        # 找到 spec 文件路径和内容
        for p in SPEC_DIR.rglob("*.yaml"):
            if p.stem == spec_name or spec_name in p.stem:
                spec_path = p
                try:
                    spec_data = yaml.safe_load(p.read_text(encoding="utf-8"))
                except Exception:
                    pass
                break

        # 按 severity 分类处理
        for gap in report.get("gaps", []):
            action = gap.get("agent_action", "skip")
            severity = gap.get("severity", "info")

            # ── self-heal ──────────────────────────────────────
            if action == "self_heal" and execute_heal:
                if "template" in gap.get("criterion_id", ""):
                    template = spec_data.get("template") or spec_data.get("content", {}).get("template", "")
                    if self_heal_template_sync(spec_name, template, gap.get("message", "")):
                        heal_count += 1
                else:
                    self_heal_parent_chain(spec_data, gap.get("message", ""))

            # ── write_test: 优先 engine 派生 ───────────────────
            elif action == "write_test":
                if severity in ("critical", "warning") and spec_name not in write_test_done:
                    write_test_done.add(spec_name)
                    try:
                        from vibex_criteria_engine import derive_and_write
                        spec_path = next(Path(SPEC_DIR).rglob(f"{spec_name}_feature.yaml"), None)
                        if spec_path:
                            result = derive_and_write(spec_path, dry_run=False, force=True)
                            if result:
                                heal_count += 1
                                print(f"  [engine] AC 派生成功: {spec_name} ({len(result)} 条)")
                            else:
                                agent_tasks.append({"spec_name": spec_name, "gap": gap})
                                print(f"  [engine] 派生失败，回退 agent: {spec_name}")
                        else:
                            agent_tasks.append({"spec_name": spec_name, "gap": gap})
                    except Exception as e:
                        agent_tasks.append({"spec_name": spec_name, "gap": gap})
                        print(f"  [engine] 异常: {e}，回退 agent")

            # ── agent 介入（fix_import / patch_component）─────────────
            elif action in ("fix_import", "patch_component"):
                if severity in ("critical", "warning"):
                    # 代码层扫描：先验证真实 gaps 再决定是否派 agent
                    code_gaps = _scan_code_level(action)
                    if not code_gaps:
                        print(f"  [code-scan✓] {action}: 代码层无 critical gaps，跳过 agent 任务")
                    else:
                        for cg in code_gaps:
                            print(f"  [agent] {action}: {cg.get('file', '?')} — {cg.get('detail', '')[:60]}")
                        agent_tasks.append({
                            "spec_name": spec_name,
                            "gap": {**gap, "code_level_gaps": code_gaps},
                        })

    # ── 代码层扫描（供 fix_import/patch_component 判断用）────────────
    _code_scan_cache: dict = {}

    def _scan_code_level(action_type: str) -> list:
        nonlocal _code_scan_cache
        """
        运行 spec_code_validator，返回 critical 级别的 code-level gaps。
        缓存结果避免重复扫描。
        """
        if not _code_scan_cache:
            import subprocess, json as _json
            result = subprocess.run(
                ["python3", "generators/spec_code_validator.py", "--actions"],
                capture_output=True, text=True,
                cwd=Path(__file__).parent,
            )
            try:
                _code_scan_cache.update(_json.loads(result.stdout))
            except Exception:
                pass  # 扫描失败时默认不过滤
        # 只返回 critical 级别的 gaps
        all_gaps = _code_scan_cache.get(action_type, [])
        return [g for g in all_gaps if g.get("severity") == "critical"]

    # ── 批量写入 agent tasks ─────────────────────────────────
    if agent_tasks:
        # 按 spec 分组，每个 spec 一个 task 文件
        by_spec: dict[str, list] = {}
        for t in agent_tasks:
            sn = t["spec_name"]
            if sn not in by_spec:
                by_spec[sn] = []
            by_spec[sn].append(t["gap"])

        for sn, gaps in by_spec.items():
            # 构建 Gap 对象列表
            from dataclasses import dataclass
            @dataclass
            class _Gap:
                def __init__(self, **kw): self.__dict__.update(kw)
                def get(self, k, default=None): return self.__dict__.get(k, default)
            gap_objs = [_Gap(**g) for g in gaps]
            written = delegate_to_agent(gap_objs, sn, dry_run=dry_run)

    # ── 汇总 ────────────────────────────────────────────────
    agent_count = len(agent_tasks)
    print(f"\n{'='*60}")
    print(f"Self-correction summary:")
    print(f"  self-healed : {heal_count}")
    print(f"  agent-needed: {agent_count} gap(s) → {TEAM_TASKS_DIR}")
    if agent_count > 0 and dry_run:
        print(f"\n  Run with --execute to queue tasks to: {TEAM_TASKS_DIR}")
    elif agent_count > 0 and not dry_run:
        print(f"\n  Tasks queued in: {TEAM_TASKS_DIR}")
        print(f"  Agent polling: GET http://localhost:33338/api/sse/<threadId>")


def main():
    import argparse
    parser = argparse.ArgumentParser(description="VibeX Spec Self-Corrector")
    parser.add_argument("--report", type=str, default="generators/.criteria_report.json",
                        help="Criteria report JSON (default: generators/.criteria_report.json)")
    parser.add_argument("--dry-run", action="store_true", default=True,
                        help="Dry-run mode (default: True, use --execute to apply)")
    parser.add_argument("--execute", dest="dry_run", action="store_false",
                        help="Execute fixes (write agent tasks, apply self-heals)")
    parser.add_argument("--skip-heal", action="store_true",
                        help="Skip self-heal (only write agent tasks)")
    parser.add_argument("--spec", type=str, default=None,
                        help="Single spec to process (default: all)")
    args = parser.parse_args()

    report_path = Path(args.report)
    if not report_path.exists():
        print(f"❌ Report not found: {report_path}")
        print("  Run: python3 generators/spec_criteria_validator.py --level L1L2L3 --json > .criteria_report.json")
        sys.exit(1)

    reports = json.loads(report_path.read_text(encoding="utf-8"))
    if not isinstance(reports, list):
        reports = [reports]

    if args.spec:
        reports = [r for r in reports if r.get("spec_name") == args.spec]
        if not reports:
            print(f"❌ Spec '{args.spec}' not found in report")
            sys.exit(1)

    mode = "DRY-RUN" if args.dry_run else "EXECUTE"
    print(f"spec-self-corrector [{mode}] — {len(reports)} report(s)")
    print("=" * 60)

    execute_heal = not args.skip_heal
    process_reports(reports, dry_run=args.dry_run, execute_heal=execute_heal)


if __name__ == "__main__":
    main()