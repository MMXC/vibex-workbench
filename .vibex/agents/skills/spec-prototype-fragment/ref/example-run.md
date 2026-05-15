# 完整运行示例

## 场景

生成 `MOD-build-panel` 的派生原型。

## Step 1：标注定位

在 MVP 中找到 `.build-group`（titlebar trail 区，data-vibex-title="Build"）。

用户确认 `.build-group`。

## Step 2：生成脚本

```bash
python scripts/wc-gen.py \
  --spec-name MOD-build-panel \
  --selector ".build-group" \
  --output ../prototypes/MOD-build-panel.html
```

脚本输出：
```
Replaced selector '.build-group' with <wc-MOD-build-panel>
Generated: ../prototypes/MOD-build-panel.html (45230 bytes)
```

## Step 3：验证

```bash
python scripts/validate.py ../prototypes/MOD-build-panel.html
```

脚本输出：
```
  ✓ MVP 全量复制
  ✓ WC 定义
  ✓ shadow DOM
  ✓ WC 元素
  ✓ 四态 panels
  ✓ 初始 hidden
  ✓ CSS 变量使用
  ✓ 无意外 hidden
8/8 检查通过
```

## Step 4：回写 Spec

```yaml
prototype:
  file: ".vibex/prototypes/MOD-build-panel.html"
  validates: []
  status: derived
```

## Step 5：人工确认

向用户输出原型路径和四态流程，确认后提交。
