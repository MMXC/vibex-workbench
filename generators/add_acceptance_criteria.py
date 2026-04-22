#!/usr/bin/env python3
"""
批量为 feature specs 补充 acceptance_criteria（Gherkin 格式）。
从 io_contract + user_stories 蒸馏而来。
用于 spec-first 自举闭环的 write_test 阶段。
"""
import yaml
from pathlib import Path

FEATURE_SPECS = [
    ("specs/feature/code-gen-panel/code-gen-panel_feature.yaml", {
        "module": "MOD-code-generator",
        "description": "代码生成面板",
        "io_contract": {
            "input": "用户 prompt + 上下文章用 spec",
            "output": "生成代码片段 + 语言标签",
            "boundary": "只做代码生成，不做执行/测试"
        },
        "behaviors": [
            "用户输入 prompt → 调用代码生成服务",
            "生成代码后高亮显示，支持复制",
            "上下文章用 spec 内容注入生成上下文"
        ]
    }),
    ("specs/feature/routing-panel/routing-panel_feature.yaml", {
        "module": "MOD-workbench-shell",
        "description": "路由面板",
        "io_contract": {
            "input": "当前对话状态 + 面板可见性",
            "output": "路由视图（对话/Canvas/spec 切换）",
            "boundary": "只做展示和路由触发，不做状态管理"
        },
        "behaviors": [
            "显示当前路由路径和面包屑",
            "点击路径节点切换视图",
            "面板展开/收起动画流畅"
        ]
    }),
    ("specs/feature/canvas-renderer/canvas-renderer_feature.yaml", {
        "module": "MOD-dsl-visualizer",
        "description": "Canvas 渲染器",
        "io_contract": {
            "input": "CanvasNode[] + 连接线数据",
            "output": "Canvas 2D/SVG 渲染结果",
            "boundary": "只做渲染，不做数据处理"
        },
        "behaviors": [
            "接收 CanvasNode[] 渲染节点",
            "连接线按时间顺序排列",
            "节点颜色按 node_type 区分"
        ]
    }),
    ("specs/feature/workbench-shell/workbench-shell_feature.yaml", {
        "module": "MOD-workbench-shell",
        "description": "工作台 Shell 框架",
        "io_contract": {
            "input": "面板布局配置 + 主题设置",
            "output": "完整工作台界面（所有面板）",
            "boundary": "框架层，不含具体业务逻辑"
        },
        "behaviors": [
            "初始化所有面板并挂载到正确位置",
            "响应主题变更（亮/暗）",
            "面板拖拽 resize 后状态持久化"
        ]
    }),
    ("specs/feature/workbench-shell/workbench-ide-chrome_feature.yaml", {
        "module": "MOD-workbench-shell",
        "description": "IDE Chrome（标题栏/标签栏/工具栏）",
        "io_contract": {
            "input": "当前打开的文件/标签 + 工作区路径",
            "output": "Chrome 区域渲染（标题/标签/按钮）",
            "boundary": "只做展示和交互触发"
        },
        "behaviors": [
            "显示当前文件路径和标签",
            "标签点击切换激活文件",
            "工具栏按钮响应点击事件"
        ]
    }),
    ("specs/feature/workbench-shell/workbench-conversation_feature.yaml", {
        "module": "MOD-workbench-shell",
        "description": "对话区域",
        "io_contract": {
            "input": "消息历史 + 当前输入",
            "output": "渲染后的对话消息列表",
            "boundary": "只做展示，消息发送由父组件处理"
        },
        "behaviors": [
            "渲染消息列表（用户/助手/系统）",
            "新消息自动滚动到底部",
            "消息支持 Markdown 渲染"
        ]
    }),
    ("specs/feature/workbench-shell/workbench-layout_resize_feature.yaml", {
        "module": "MOD-workbench-shell",
        "description": "布局 resize/drag",
        "io_contract": {
            "input": "拖拽位置 + 面板尺寸约束",
            "output": "更新后的面板尺寸",
            "boundary": "只计算尺寸，不做持久化"
        },
        "behaviors": [
            "拖拽分隔条调整相邻面板大小",
            "面板尺寸最小值约束生效",
            "resize 后触发父组件状态更新"
        ]
    }),
    ("specs/feature/spec-editor/spec-editor_feature.yaml", {
        "module": "MOD-spec-engine",
        "description": "Spec 编辑器",
        "io_contract": {
            "input": "spec YAML 内容 + 光标位置",
            "output": "渲染后的编辑器视图 + 自动补全",
            "boundary": "只做编辑展示，保存由父组件处理"
        },
        "behaviors": [
            "YAML 语法高亮显示",
            "自动补全 spec 字段（given/when/then 等）",
            "实时期务校验（parent 链、层级结构）"
        ]
    }),
]


def build_acceptance_criteria(meta: dict) -> list:
    """从 io_contract + behaviors 蒸馏 Gherkin AC。"""
    io = meta.get("io_contract", {})
    desc = meta.get("description", "")
    behaviors = meta.get("behaviors", [])

    acs = []

    # AC-1: 核心输入输出
    if io.get("input") and io.get("output"):
        acs.append({
            "id": "AC-1",
            "given": f"用户在 {desc}区域",
            "when": "触发核心行为",
            "then": f"输入「{io['input']}」正确映射到输出「{io['output']}」"
        })

    # AC-2: 行为流
    if behaviors:
        acs.append({
            "id": "AC-2",
            "given": f"用户在 {desc}区域",
            "when": f"执行「{behaviors[0]}」",
            "then": f"预期结果符合「{behaviors[0]}」描述，且 UI 反馈正确"
        })

    # AC-3: 边界约束
    if io.get("boundary"):
        acs.append({
            "id": "AC-3",
            "given": "系统在边界条件下",
            "when": f"边界：「{io['boundary']}」",
            "then": "系统不越界，仍保持正常展示或给出明确错误"
        })

    # AC-4: 视图更新
    acs.append({
        "id": "AC-4",
        "given": f"用户在 {desc}区域且有数据状态",
        "when": "数据或状态变更",
        "then": "视图在 300ms 内正确更新，无闪烁或数据丢失"
    })

    return acs


def add_acceptance_criteria_to_spec(path: Path, meta: dict) -> bool:
    """向 spec YAML 添加 acceptance_criteria（保留注释，只追加到文件末尾）。"""
    try:
        content = path.read_text(encoding="utf-8")
        data = yaml.safe_load(content)
    except Exception as e:
        print(f"  ❌ 读取失败: {path} — {e}")
        return False

    # 已存在则跳过
    if data.get("acceptance_criteria"):
        print(f"  ⏭️  已有 AC，跳过: {path.stem}")
        return False

    acs = build_acceptance_criteria(meta)

    # 用 yaml block 风格追加到文件末尾（保留原有注释和格式）
    ac_yaml = yaml.dump(
        {"acceptance_criteria": acs},
        allow_unicode=True,
        default_flow_style=False,
        sort_keys=False
    )
    # acceptance_criteria 是顶级 key，不需要额外缩进（直接追加到文件末尾）
    ac_block = ac_yaml.strip()

    new_content = content.rstrip() + "\n" + ac_block + "\n"
    path.write_text(new_content, encoding="utf-8")

    # 验证写入后 YAML 仍可解析
    try:
        yaml.safe_load(new_content)
    except Exception as e:
        print(f"  ❌ 写入后 YAML 解析失败: {path} — {e}")
        return False

    print(f"  ✅ 添加 AC({len(acs)}条): {path.stem}")
    return True


def main():
    print("批量添加 acceptance_criteria")
    print("=" * 50)
    for spec_path, meta in FEATURE_SPECS:
        p = Path(spec_path)
        if p.exists():
            add_acceptance_criteria_to_spec(p, meta)
        else:
            print(f"  ❌ 文件不存在: {spec_path}")
    print("\n完成！运行 make spec-criteria 验证。")


if __name__ == "__main__":
    main()
