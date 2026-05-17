# Cal4God2 设计规范

## 1. 技术栈

| 层级 | 技术 | 说明 |
|------|------|------|
| 前端框架 | Next.js 14 (App Router) | React 18 + TypeScript |
| UI 组件 | Radix UI + shadcn/ui | 组件库 |
| 样式 | Tailwind CSS | 原子化 CSS |
| 状态管理 | React Context + useReducer | 轻量级方案 |
| 数据持久化 | localStorage | 本地存储 |
| HTTP 客户端 | 原生 fetch | 无需额外依赖 |

## 2. 数据模型

### 2.1 词条模型 (StatValue)

所有物品共享的基础属性结构：

```typescript
interface StatValue {
  gj?: number;   // 攻击
  bs?: number;   // 冰属性伤害
  hs?: number;   // 火属性伤害
  ds?: number;   // 毒属性伤害
  ls?: number;   // 雷属性伤害
  bm?: number;   // 冰属性抗性
  dm?: number;   // 毒属性抗性
  lm?: number;   // 雷属性抗性
  hm?: number;   // 火属性抗性
  hx?: number;   // 暗属性
  bjl?: number;  // 暴击率
  bjsh?: number; // 暴击伤害
  qsxsh?: number;// 全属性伤害
  yczs?: number; // 远古之力
  dbzs?: number; // 对Boss增伤
  zzsh?: number; // 最终伤害
  pf?: number;   // 破防
  ct?: number;   // 穿透
  jn?: number;   // 技能伤害
  jk?: number;   // 减伤
}
```

### 2.2 物品类型

| 类型 | 代码 | 说明 | 最大数量 |
|------|------|------|----------|
| 主卡 | zk | 角色卡片 | 12 |
| 装备 | zb | 武器/防具/饰品 | 10 |
| 套装 | tz | 套装组合 | - |
| 羁绊 | jb | 角色羁绊 | 29 |
| 赋能 | fn | 天赋赋能 | 9 |
| 远古词条 | hy | 黄印词条 | 7 |
| 符文 | fw | 单符文 | 20 |
| 符文之语 | fwzy | 符文组合 | 30 |

### 2.3 组合激活规则

#### 套装 (tz)
- 通过 `num` 字段判断激活数量
- 值 2/4/6 表示装备该套装达到对应件数时激活

#### 羁绊 (jb)
- 通过 `num` 字段判断羁绊人数
- `first/second/third/forth` 字段表示各段效果
- 羁绊人数达到 `num` 时激活对应段效果

## 3. 数据源

### 3.1 JSON 文件映射

| 实体 | 文件路径 | API 端点 |
|------|----------|----------|
| 主卡 | `public/data/zk.json` | `/api/zk` |
| 装备 | `public/data/zb.json` | `/api/zb` |
| 套装 | `public/data/tz.json` | `/api/tz` |
| 羁绊 | `public/data/jb.json` | `/api/jb` |
| 赋能 | `public/data/fn.json` | `/api/fn` |
| 远古词条 | `public/data/hy.json` | `/api/hy` |
| 符文 | `public/data/fw.json` | `/api/fw` |
| 符文之语 | `public/data/fwzy.json` | `/api/fwzy` |

### 3.2 数据格式

所有 JSON 文件遵循统一格式：

```json
{
  "code": 0,
  "success": "true",
  "list": [
    { /* 物品对象 */ }
  ]
}
```

## 4. 适配层

### 4.1 适配器接口

```typescript
interface DataAdapter {
  loadAll(): Promise<Record<ItemCategory, AnyItem[]>>;
  loadCategory(category: ItemCategory): Promise<AnyItem[]>;
  query(category: ItemCategory, filter?: QueryFilter): Promise<AnyItem[]>;
  getById(category: ItemCategory, id: number): Promise<AnyItem | null>;
  search(category: ItemCategory, keyword: string): Promise<AnyItem[]>;
  calculateCombos(build: Build): ComboEffect[];
  exportBuild(build: Build, format: ExportFormat): string;
  importBuild(data: string, format: ExportFormat): Build | null;
}
```

### 4.2 适配器实现

| 适配器 | 说明 | 使用场景 |
|--------|------|----------|
| HttpDataAdapter | HTTP 请求加载远程数据 | 正式环境 |
| LocalStorageAdapter | localStorage 缓存 + HTTP 降级 | 开发/离线 |

## 5. 导入导出

### 5.1 导出格式

| 格式 | 文件扩展 | 说明 |
|------|----------|------|
| JSON | `.json` | 人类可读，包含完整元数据 |
| Base64 | `.cal4god2` | 压缩编码，适合复制分享 |

### 5.2 导出包结构

```typescript
interface ExportPackage {
  version: string;      // 版本号 "1.0.0"
  exportedAt: string;    // ISO 时间戳
  build: Build;         // BUILD 方案
  metadata?: {
    author?: string;    // 作者
    description?: string;// 描述
    tags?: string[];    // 标签
  };
}
```

## 6. 组件边界

### 6.1 业务组件

| 组件 | 文件路径 | 职责 |
|------|----------|------|
| Cal | `src/components/component/cal.tsx` | 角色卡展示 |
| Cal2 | `src/components/component/cal2.tsx` | 角色卡 V2 展示 |
| Zb | `src/components/component/zb.tsx` | 装备选择 |
| Tz | `src/components/component/tz.tsx` | 套装展示 |
| Jb | `src/components/component/jb.tsx` | 羁绊展示 |
| Fn | `src/components/component/fn.tsx` | 赋能选择 |
| Hy | `src/components/component/hy.tsx` | 远古词条 |
| Fw | `src/components/component/fw.tsx` | 符文选择 |
| Fwzy | `src/components/component/fwzy.tsx` | 符文之语 |

### 6.2 Context 边界

| Context | 职责 |
|---------|------|
| UserSelectionsContext | 用户选择状态管理 |
| RoleContext | 角色数据与计算逻辑 |

## 7. 验证规则

### 7.1 选择限制

| 类型 | 最大数量 | 验证规则 |
|------|----------|----------|
| 主卡 | 12 | zkSelection.length <= 12 |
| 装备 | 10 | zbSelection.length <= 10 |
| 装备类型 | 1 | 同 type 只允许一件 |

### 7.2 组合激活

| 组合类型 | 触发条件 | 效果 |
|----------|----------|------|
| 套装 | 装备同套装 2/4/6 件 | 激活对应词条加成 |
| 羁绊 | 选中羁绊中的角色卡 | 激活羁绊效果 |
| 符文之语 | 收集齐所需符文 | 激活符文之语效果 |

## 8. 文件结构

```
cal4god2/
├── public/
│   ├── data/              # 静态 JSON 数据
│   │   ├── zk.json
│   │   ├── zb.json
│   │   └── ...
│   └── assets/            # 图片资源
│       ├── zk/
│       ├── zb/
│       └── ...
├── src/
│   ├── app/               # Next.js App Router
│   │   └── pages/
│   │       └── Simulator.tsx
│   ├── components/
│   │   ├── component/     # 业务组件
│   │   └── ui/           # shadcn/ui 组件
│   ├── contexts/         # React Context
│   ├── providers/        # Provider 封装
│   └── services/         # API 服务
├── .vibex/
│   ├── design/            # 设计规范
│   │   ├── DESIGN.md
│   │   ├── types.ts
│   │   └── adapter.ts
│   ├── specs/            # Spec 文件
│   └── prototypes/       # 原型文件
└── package.json
```
