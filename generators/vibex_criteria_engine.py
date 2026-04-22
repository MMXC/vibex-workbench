#!/usr/bin/env python3
"""
vibex-criteria-engine: 从 spec YAML 结构化数据派生 acceptance_criteria（Gherkin 格式）。

设计原则（goskill 启发）：
  - Criteria 从结构化数据派生，不是 LLM 推理
  - 派生规则确定性、可重复、可测试
  - 三层数据源：io_contract > user_stories > behavior
  - 只在派生失败时 fallback 到 agent

派生策略：
  1. user_stories → Gherkin（直接映射 as_a→given, i_want→when, so_that→then）
  2. io_contract + behavior → Gherkin（解析输入/触发/输出/边界）
  3. capabilities → Gherkin（从能力描述生成边界条件）
"""

import re
import yaml
from pathlib import Path
from typing import Optional


# ── 解析工具 ─────────────────────────────────────────────────────────

def parse_behavior_steps(behavior_text: str) -> list[dict]:
    """
    解析 io_contract.behavior 文本为步骤列表。
    支持格式：
      1. 步骤文字
      2. - 步骤文字
      3. ### 标题\n  步骤文字
      4. Mermaid视图：\n  1. xxx\n  2. xxx\n    Canvas视图：\n  1. xxx
    （双视图格式）
    返回: [{"type": "header"|"step"|"substep", "text": str, "view": str|None}, ...]
    """
    if not behavior_text or not isinstance(behavior_text, str):
        return []

    steps = []
    lines = behavior_text.strip().splitlines()
    current_view = None

    for line in lines:
        stripped = line.strip()
        if not stripped:
            continue

        # 双视图标题: "Mermaid视图：" / "Canvas视图（只读）："
        view_match = re.match(r'([^\[（）()\n]+视图[^:：]*)[：:]', stripped)
        if view_match:
            current_view = view_match.group(1).strip()
            steps.append({"type": "header", "text": current_view, "view": current_view})
            continue

        # 编号步骤: "1. xxx" / "2. xxx"
        numbered = re.match(r'^(\d+)[.．、](.*)', stripped)
        if numbered:
            steps.append({
                "type": "step",
                "index": int(numbered.group(1)),
                "text": numbered.group(2).strip(),
                "view": current_view
            })
            continue

        # 无编号步骤: "- xxx" / "  xxx"
        if stripped.startswith('-') or (line.startswith('  ') and stripped):
            text = stripped.lstrip('-').strip()
            steps.append({
                "type": "step",
                "text": text,
                "view": current_view
            })
            continue

        # 标题行
        if re.match(r'^#{1,3}\s', stripped) or re.match(r'^[*_]{1,2}', stripped):
            steps.append({"type": "header", "text": stripped, "view": current_view})
            continue

    return steps


def extract_sse_events(text: str) -> list[str]:
    """从文本中提取 SSE/事件名称，如 generation.started, dialog.* 等。"""
    # 匹配 xxx.yyy 或 xxx.* 格式
    events = re.findall(r'([a-zA-Z_][a-zA-Z0-9_]*\.[a-zA-Z_*]+)', text)
    return list(dict.fromkeys(events))  # 去重保持顺序


def extract_outputs(text: str) -> list[str]:
    """从 output 字段提取输出项。"""
    if isinstance(text, list):
        return [str(item).strip('"').strip() for item in text]
    lines = str(text).strip().splitlines()
    return [line.strip().lstrip('- ').strip() for line in lines if line.strip()]


def extract_inputs(text: str) -> list[str]:
    """从 input 字段提取输入项。"""
    if isinstance(text, list):
        return [str(item).strip('"').strip() for item in text]
    lines = str(text).strip().splitlines()
    return [line.strip().lstrip('- ').strip() for line in lines if line.strip()]


# ── AC 派生器 ─────────────────────────────────────────────────────────

class CriteriaDeriver:
    """从 spec YAML 结构化数据派生 acceptance_criteria。"""

    def __init__(self, spec_data: dict):
        self.data = spec_data
        self.meta = spec_data.get("meta", {})
        self.content = spec_data.get("content", {})
        self.io = self.content.get("io_contract", {})
        self.behavior = self.io.get("behavior", "")
        self.user_stories = self.content.get("user_stories", [])
        self.capabilities = self.content.get("capabilities", [])
        self.spec_name = spec_data.get("spec", {}).get("name", "unknown")
        self._steps = None

    @property
    def steps(self) -> list[dict]:
        if self._steps is None:
            self._steps = parse_behavior_steps(self.behavior)
        return self._steps

    def derive(self) -> list[dict]:
        """
        派生 acceptance_criteria。
        优先用 user_stories（直接映射），其次用 behavior + io_contract。
        """
        acs = []

        # 策略 1: user_stories → Gherkin
        if self.user_stories:
            acs.extend(self._from_user_stories())

        # 策略 2: io_contract.behavior → Gherkin
        if len(acs) < 4 and self.steps:
            acs.extend(self._from_behavior(max_additional=4 - len(acs)))

        # 策略 3: capabilities → Gherkin（边界/优先级）
        if len(acs) < 4 and self.capabilities:
            acs.extend(self._from_capabilities(max_additional=4 - len(acs)))

        # 策略 4: io_contract 基础派生（兜底）
        if len(acs) < 3:
            acs.extend(self._from_io_contract(max_additional=3 - len(acs)))

        # 策略 5: description + vision_traceability（canvas-renderer 等）
        if len(acs) < 3:
            acs.extend(self._from_description(max_additional=3 - len(acs)))

        # 分配 AC-ID
        for i, ac in enumerate(acs, 1):
            if "id" not in ac or not ac["id"]:
                ac["id"] = f"AC-{i}"

        return acs[:8]  # 最多 8 条，避免过多

    # ── 策略 1: user_stories ──────────────────────────────────────

    def _from_user_stories(self) -> list[dict]:
        acs = []
        for us in self.user_stories[:5]:
            us_id = us.get("id", "")
            as_a = us.get("as_a", "")
            i_want = us.get("i_want", "")
            so_that = us.get("so_that", "")

            # 清理文本
            i_want = re.sub(r'^我|^系统|^用户|^开发者|^spec 作者\s*', '', i_want).strip()

            given = self._build_given(as_a)
            then = self._build_then(so_that, i_want)
            when = self._build_when(i_want)

            acs.append({
                "given": given,
                "when": when,
                "then": then,
                "_source": f"user_story:{us_id}",
            })
        return acs

    def _build_given(self, as_a: str) -> str:
        """从角色描述构建 given 前置条件。"""
        area = self._spec_area()
        # 提取角色关键词
        role = as_a
        for r in ["spec 作者", "开发者", "用户", "系统管理员", "系统"]:
            if r in as_a:
                role = r
                break
        return f"{role}在 {area}"

    def _build_when(self, i_want: str) -> str:
        """从用户意图构建 when 触发条件。"""
        # i_want 通常是"我/系统 + 动作 + 对象"，直接截取前 60 字符
        # 去掉句末标点
        text = re.sub(r'[，。、；：\n].*$', '', i_want.strip())
        return text[:60]

    def _build_then(self, so_that: str, i_want: str) -> str:
        """从目的描述构建 then 期望结果。"""
        if so_that:
            # 清理句末标点和无关文字
            text = re.sub(r'[，。、；：\n].*$', '', so_that.strip())
            return text[:80]
        # 兜底：i_want 本身作为期望
        return re.sub(r'^[我系统用户开发者spec 作者\s]+', '', i_want)[:80]

    # ── 策略 2: behavior ───────────────────────────────────────────

    def _from_behavior(self, max_additional: int = 4) -> list[dict]:
        """
        从 io_contract.behavior 步骤列表派生 AC。
        策略：
          - 第一条 AC: 输入 → 初始状态（given: 接收输入，when: 触发，then: 第一个输出）
          - 后续 AC: 逐步骤映射（when: 步骤文字, then: 输出变化）
          - viewMode 切换: 单独 AC
        """
        acs = []
        steps = self.steps

        if not steps:
            return acs

        # 提取 viewModes
        views = list(dict.fromkeys(s.get("view") for s in steps if s.get("view") and "视图" in s.get("view", "")))

        # 找到第一条步骤（通常是最外层的 trigger）
        top_steps = [s for s in steps if s.get("type") == "step" and s.get("view") is None]
        if not top_steps:
            top_steps = steps[:6]

        if not top_steps:
            return acs

        # AC-1: 接收触发
        first = top_steps[0]
        trigger_events = extract_sse_events(first.get("text", ""))
        if trigger_events:
            acs.append({
                "given": f"系统在运行{self._spec_area()}",
                "when": f"接收到 {trigger_events[0]} 事件",
                "then": f"系统开始处理，状态变为 processing",
                "_source": "behavior:trigger",
            })
        else:
            step_text = first.get("text", "")[:50]
            acs.append({
                "given": f"用户在{self._spec_area()}",
                "when": step_text,
                "then": f"系统响应 {step_text.split('→')[0] if '→' in step_text else step_text[:30]}",
                "_source": "behavior:trigger",
            })

        # 后续步骤 → AC
        for step in top_steps[1:max_additional]:
            text = step.get("text", "")
            if not text:
                continue
            events = extract_sse_events(text)
            outputs = extract_outputs(text)

            if events:
                then = f"发送 {events[0]} 事件"
                if len(events) > 1:
                    then += f"，触发后续流程"
                acs.append({
                    "given": f"系统在处理中",
                    "when": text[:60],
                    "then": then,
                    "_source": "behavior:step",
                })
            elif outputs:
                acs.append({
                    "given": f"系统在处理中",
                    "when": text[:60],
                    "then": outputs[0][:60] if isinstance(outputs, list) else str(outputs)[:60],
                    "_source": "behavior:output",
                })

        # viewMode 切换（如果有多个视图）
        if len(views) >= 2 and len(acs) < max_additional + 2:
            view_names = [v.replace("视图", "").replace("（只读）", "") for v in views[:2]]
            acs.append({
                "given": f"用户在{self._spec_area()}（任一视图）",
                "when": f"切换 {view_names[0]} ↔ {view_names[1]}",
                "then": f"视图在 500ms 内完成切换，数据源正确切换，无状态丢失",
                "_source": "behavior:viewmode",
            })

        return acs[:max_additional]

    # ── 策略 3: capabilities ──────────────────────────────────────

    def _from_capabilities(self, max_additional: int = 3) -> list[dict]:
        acs = []
        for cap in self.capabilities[:max_additional]:
            name = cap.get("name", "")
            desc = cap.get("description", "")
            priority = cap.get("priority", "medium")

            if not name:
                continue

            # 提取能力核心词
            trigger = re.match(r'([^（]+)', name)
            trigger_text = trigger.group(1).strip() if trigger else name

            # high/critical → 性能/正确性期望
            if priority in ("critical", "high"):
                acs.append({
                    "given": f"用户在执行 {trigger_text}",
                    "when": f"触发 {trigger_text}",
                    "then": f"功能正常工作，结果符合预期",
                    "_source": f"capability:{cap.get('id','')}",
                })
            else:
                acs.append({
                    "given": f"用户在使用系统",
                    "when": f"触发 {trigger_text}",
                    "then": f"功能按描述正常工作",
                    "_source": f"capability:{cap.get('id','')}",
                })

        return acs

    # ── 策略 5: description + vision_traceability ─────────────────

    def _from_description(self, max_additional: int = 3) -> list[dict]:
        """
        从 description + vision_traceability 派生 AC。
        用于 canvas-renderer 等没有 io_contract 但有 vision_traceability 的 spec。
        """
        acs = []
        desc = self.content.get("description", "")
        vt = self.content.get("vision_traceability", {})
        summary = vt.get("summary", "")
        boundaries = vt.get("boundaries_vs_dsl_visualizer", {})
        acceptance = vt.get("acceptance_summary", [])

        if not desc and not summary:
            return acs

        # 从 description 提取关键词
        # "Canvas 渲染器——@xyflow/svelte 包装层，SSE 事件流 → 节点序列图"
        parts = desc.split("——")
        if len(parts) >= 2:
            component = parts[0].strip()
            capability = parts[1].strip().rstrip("。")
        else:
            component = self.spec_name
            capability = desc.strip().rstrip("。")

        # AC-1: 核心功能
        if "SSE" in capability or "事件" in capability:
            acs.append({
                "given": f"用户在 {component}",
                "when": "接收到 SSE 事件流",
                "then": "事件被映射为节点序列并渲染",
                "_source": "description:core",
            })

        # AC-2: 渲染输出
        if "节点" in capability:
            acs.append({
                "given": f"系统在运行 {component}",
                "when": "有对话行为数据到达",
                "then": "渲染为节点序列图，数据完整",
                "_source": "description:render",
            })

        # AC-3: 边界约束（从 vision_traceability）
        if boundaries:
            boundary_parts = list(boundaries.values())
            for bp in boundary_parts[:1]:
                if isinstance(bp, str) and "不" in bp:
                    not_do = re.search(r'不[做作为].{0,30}', bp)
                    if not_do:
                        acs.append({
                            "given": f"系统在 {component}",
                            "when": "与其他模块交互",
                            "then": not_do.group(0)[:60],
                            "_source": "vision_traceability:boundary",
                        })
                        break

        # AC-4: 验收摘要（从 acceptance_summary）
        if acceptance and len(acs) < max_additional:
            for a in acceptance[:1]:
                if isinstance(a, str) and len(a) > 10:
                    acs.append({
                        "given": f"用户在 {component}",
                        "when": "执行核心功能",
                        "then": a[:80],
                        "_source": "vision_traceability:acceptance",
                    })

        return acs[:max_additional]

    # ── 策略 6: io_contract 兜底 ─────────────────────────────────

    def _from_io_contract(self, max_additional: int = 3) -> list[dict]:
        acs = []
        inputs = extract_inputs(self.io.get("input", ""))
        outputs = extract_outputs(self.io.get("output", ""))
        boundary = self.io.get("boundary", "")

        if inputs and outputs:
            # AC: 输入 → 输出
            acs.append({
                "given": f"系统在运行状态",
                "when": f"接收到 {inputs[0] if inputs else '输入'}",
                "then": outputs[0][:80] if outputs else "正常处理",
                "_source": "io_contract:basic",
            })

        if boundary:
            # 从 boundary 提取"不做"约束 → then 中的"不..."
            not_do = re.findall(r'不[做作为].{0,30}', boundary)
            if not_do and len(acs) < max_additional:
                for nd in not_do[:2]:
                    acs.append({
                        "given": f"系统在运行{self._spec_area()}",
                        "when": "任何操作",
                        "then": nd[:60],
                        "_source": "io_contract:boundary",
                    })

        return acs[:max_additional]

    # ── 辅助 ─────────────────────────────────────────────────────

    def _spec_area(self) -> str:
        """从 spec name 提取区域描述。"""
        name = self.spec_name.lower()
        if "canvas" in name:
            return "DSL Canvas 区域"
        if "panel" in name or "gen" in name or "gen" in name:
            return "面板区域"
        if "routing" in name:
            return "路由面板"
        if "editor" in name or "spec-editor" in name:
            return "Spec 编辑器"
        if "shell" in name or "workbench" in name:
            return "工作台"
        if "chrome" in name:
            return "IDE Chrome 区域"
        if "conversation" in name:
            return "对话区域"
        if "resize" in name:
            return "布局调整"
        return "系统"


def spec_name_area(name: str) -> str:
    """standalone helper"""
    d = CriteriaDeriver({"spec": {"name": name}, "content": {}, "meta": {}})
    return d._spec_area()


# ── 主入口 ─────────────────────────────────────────────────────────

def derive_acceptance_criteria(spec_path: Path) -> list[dict]:
    """
    从 spec YAML 文件派生 acceptance_criteria。
    返回: list[dict] — Gherkin 格式的 AC 列表
    """
    try:
        data = yaml.safe_load(spec_path.read_text(encoding="utf-8"))
    except Exception as e:
        print(f"  ❌ 读取失败: {spec_path} — {e}")
        return []

    # 已存在则跳过
    if data.get("acceptance_criteria"):
        return data["acceptance_criteria"]

    deriver = CriteriaDeriver(data)
    acs = deriver.derive()
    return acs


def derive_and_write(spec_path: Path, dry_run: bool = False, force: bool = False) -> list[dict]:
    """
    派生 AC 并写回 spec 文件。
    dry_run=True: 只派生不写回。
    force=True: 覆盖已有 AC。
    """
    try:
        content = spec_path.read_text(encoding="utf-8")
        data = yaml.safe_load(content)
    except Exception as e:
        print(f"  ❌ 读取失败: {spec_path} — {e}")
        return []

    # 已存在则跳过（除非 force）
    if not force and data.get("acceptance_criteria"):
        print(f"  ⏭️  已有 AC({len(data['acceptance_criteria'])}条)，跳过: {spec_path.stem}")
        return data["acceptance_criteria"]

    deriver = CriteriaDeriver(data)
    acs = deriver.derive()

    if not acs:
        print(f"  ⚠️  派生失败，无 AC: {spec_path.stem}")
        return []

    print(f"  ✅ 派生 AC({len(acs)}条): {spec_path.stem}")

    if not dry_run:
        ac_yaml = yaml.dump(
            {"acceptance_criteria": acs},
            allow_unicode=True,
            default_flow_style=False,
            sort_keys=False
        )
        new_content = content.rstrip() + "\n" + ac_yaml
        spec_path.write_text(new_content, encoding="utf-8")

        # 验证
        try:
            yaml.safe_load(new_content)
        except Exception as e:
            print(f"  ❌ 写入后 YAML 解析失败，回滚: {e}")
            spec_path.write_text(content, encoding="utf-8")
            return []

    return acs


# ── CLI ──────────────────────────────────────────────────────────

def main():
    import argparse
    parser = argparse.ArgumentParser(description="vibex-criteria-engine: 从 spec 派生 acceptance_criteria")
    parser.add_argument("--spec", help="单个 spec 路径")
    parser.add_argument("--all", action="store_true", help="处理所有 feature spec")
    parser.add_argument("--dry-run", action="store_true", help="只派生不写回")
    parser.add_argument("--force", action="store_true", help="覆盖已有 AC")
    args = parser.parse_args()

    if args.spec:
        spec_path = Path(args.spec)
        if not spec_path.is_absolute():
            spec_path = Path("specs/feature") / args.spec
        derive_and_write(spec_path, dry_run=args.dry_run, force=args.force)
        return

    if args.all:
        specs_dir = Path("specs/feature")
        results = []
        for spec_path in sorted(specs_dir.rglob("*_feature.yaml")):
            acs = derive_and_write(spec_path, dry_run=args.dry_run, force=args.force)
            results.append((spec_path.stem, len(acs)))
        print(f"\n完成！共处理 {len(results)} 个 spec")
        for name, n in results:
            print(f"  {name}: {n} ACs")
        return

    # 默认: 检查所有 feature spec
    specs_dir = Path("specs/feature")
    total = 0
    missing = 0
    for spec_path in sorted(specs_dir.rglob("*_feature.yaml")):
        data = yaml.safe_load(spec_path.read_text())
        if not data.get("acceptance_criteria"):
            missing += 1
            deriver = CriteriaDeriver(data)
            acs = deriver.derive()
            print(f"  ⚠️ 缺 AC，派生 {len(acs)} 条: {spec_path.stem}")
        total += 1
    print(f"\n总计: {total} 个 feature spec，{missing} 个缺 AC")


if __name__ == "__main__":
    main()
