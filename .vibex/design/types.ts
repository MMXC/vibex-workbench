/**
 * Cal4God2 数据模型 - 词条、物品、组合模型
 * @version 1.0.0
 */

// ============================================
// 1. 词条模型 (StatModel)
// ============================================

export enum StatType {
  // 攻击类
  gj   = "gj",   // 攻击
  bs   = "bs",   // 冰属性伤害
  hs   = "hs",   // 火属性伤害
  ds   = "ds",   // 毒属性伤害
  ls   = "ls",   // 雷属性伤害
  
  // 属性抗性类
  bm   = "bm",   // 冰属性抗性
  dm   = "dm",   // 毒属性抗性
  lm   = "lm",   // 雷属性抗性
  hm   = "hm",   // 火属性抗性
  hx   = "hx",   // 暗属性
  
  // 暴击类
  bjl  = "bjl",  // 暴击率
  bjsh = "bjsh", // 暴击伤害
  
  // 增伤类
  qsxsh  = "qsxsh",  // 全属性伤害
  yczs   = "yczs",   // 远古之力
  dbzs   = "dbzs",   // 对Boss增伤
  zzsh   = "zzsh",   // 最终伤害
  dzdbzs = "dzdbzs", // 对Boss增伤(重复定义)
  
  // 特殊
  pf = "pf",  // 破防
  ct = "ct",  // 穿透
  jn = "jn",  // 技能伤害
  jk = "jk"   // 减伤
}

// 词条元数据
export const STAT_META: Record<string, { label: string; unit: string; category: string }> = {
  gj:    { label: "攻击",       unit: "%",      category: "attack" },
  bs:    { label: "冰属性伤害", unit: "%",      category: "elemental" },
  hs:    { label: "火属性伤害", unit: "%",      category: "elemental" },
  ds:    { label: "毒属性伤害", unit: "%",      category: "elemental" },
  ls:    { label: "雷属性伤害", unit: "%",      category: "elemental" },
  bm:    { label: "冰属性抗性", unit: "%",      category: "resistance" },
  dm:    { label: "毒属性抗性", unit: "%",      category: "resistance" },
  lm:    { label: "雷属性抗性", unit: "%",      category: "resistance" },
  hm:    { label: "火属性抗性", unit: "%",      category: "resistance" },
  hx:    { label: "暗属性",     unit: "%",      category: "elemental" },
  bjl:   { label: "暴击率",     unit: "%",      category: "crit" },
  bjsh:  { label: "暴击伤害",   unit: "%",      category: "crit" },
  qsxsh: { label: "全属性伤害", unit: "%",      category: "damage" },
  yczs:  { label: "远古之力",   unit: "%",      category: "damage" },
  dbzs:  { label: "对Boss增伤", unit: "%",      category: "damage" },
  zzsh:  { label: "最终伤害",   unit: "%",      category: "damage" },
  pf:    { label: "破防",       unit: "%",      category: "special" },
  ct:    { label: "穿透",       unit: "%",      category: "special" },
  jn:    { label: "技能伤害",   unit: "%",      category: "special" },
  jk:    { label: "减伤",       unit: "%",      category: "special" }
};

// 词条值对象
export interface StatValue {
  gj?: number; bs?: number; hs?: number; ds?: number; ls?: number;
  bm?: number; dm?: number; lm?: number; hm?: number; hx?: number;
  bjl?: number; bjsh?: number;
  qsxsh?: number; yczs?: number; dbzs?: number; zzsh?: number; dzdbzs?: number;
  pf?: number; ct?: number; jn?: number; jk?: number;
}

// ============================================
// 2. 物品类型枚举
// ============================================

export enum ItemCategory {
  zk   = "zk",    // 主卡
  zb   = "zb",    // 装备
  tz   = "tz",    // 套装
  jb   = "jb",    // 羁绊
  fn   = "fn",    // 赋能
  hy   = "hy",    // 远古词条
  fw   = "fw",    // 符文
  fwzy = "fwzy",  // 符文之语
  yg   = "yg",    // 远古装备
  jn   = "jn"     // 技能
}

// ============================================
// 3. 物品模型
// ============================================

// 基础物品
export interface BaseItem {
  id: number;
  name: string;
  pic: string;
  pre?: string;
  sx: StatValue;
}

// 主卡 (zk)
export interface ZkItem extends BaseItem {
  kind: "进攻" | "灵能" | "防御" | "辅助";
  quality: "至臻" | "史诗" | "稀有" | "普通";
  type: "神话" | "传说" | "稀有";
  zd: string;   // 主动技能
  cz: string;   // 成长公式
  ls: string;   // 被动技能
}

// 装备 (zb)
export interface ZbItem extends BaseItem {
  num: number;     // 数量
  sj_num: number;  // 随机词条数
  type: "轻武" | "重武" | "饰品" | "混沌" | "遗骨" | "活物" | "神话";
}

// 套装 (tz)
export interface TzItem extends BaseItem {
  num: number;  // 触发数量 (0=被动, 2/4/6=件数)
}

// 羁绊 (jb)
export interface JbItem extends BaseItem {
  num: number;
  level: "一级" | "二级" | "三级" | "四级";
  first: StatValue;
  second: StatValue;
  third: StatValue;
  forth: StatValue;
}

// 符文 (fw)
export interface FwItem extends BaseItem {
  type: string;
}

// 符文之语 (fwzy)
export interface FwzyItem extends BaseItem {
  num: number;
  level: number;
}

// 赋能 (fn)
export interface FnItem extends BaseItem {
  type: "fn";
}

// 远古词条 (hy)
export interface HyItem extends BaseItem {
  num: number;
  type: "hy";
  ct?: number;
  jn?: number;
}

// 统一物品类型
export type AnyItem = ZkItem | ZbItem | TzItem | JbItem | FwItem | FwzyItem | FnItem | HyItem;

// ============================================
// 4. 组合模型
// ============================================

export enum ComboType {
  tz   = "tz",    // 套装
  jb   = "jb",    // 羁绊
  fwzy = "fwzy"   // 符文之语
}

export interface ComboEffect {
  type: ComboType;
  id: number;
  name: string;
  required: number;   // 触发所需数量
  current: number;    // 当前数量
  isActive: boolean;
  stats: StatValue;
}

// ============================================
// 5. BUILD方案模型
// ============================================

export interface Build {
  id: string;
  name: string;
  createdAt: string;
  updatedAt: string;
  
  zkSelection: ZkItem[];
  zbSelection: ZbItem[];
  fnSelection: FnItem[];
  jbSelection: JbItem[];
  hySelection: HyItem[];
  fwSelection: FwItem[];
  fwzySelection: FwzyItem[];
  
  totalStats: StatValue;
  activeCombos: ComboEffect[];
}

// ============================================
// 6. DataCenter 数据源配置
// ============================================

export interface DataSource {
  category: ItemCategory;
  endpoint: string;
  file: string;
  model: string;
}

export const DATA_SOURCES: DataSource[] = [
  { category: ItemCategory.zk,   endpoint: "/api/zk",   file: "public/data/zk.json",   model: "ZkItem" },
  { category: ItemCategory.zb,   endpoint: "/api/zb",   file: "public/data/zb.json",   model: "ZbItem" },
  { category: ItemCategory.tz,   endpoint: "/api/tz",   file: "public/data/tz.json",   model: "TzItem" },
  { category: ItemCategory.jb,   endpoint: "/api/jb",   file: "public/data/jb.json",   model: "JbItem" },
  { category: ItemCategory.fn,   endpoint: "/api/fn",   file: "public/data/fn.json",   model: "FnItem" },
  { category: ItemCategory.hy,   endpoint: "/api/hy",   file: "public/data/hy.json",   model: "HyItem" },
  { category: ItemCategory.fw,   endpoint: "/api/fw",   file: "public/data/fw.json",   model: "FwItem" },
  { category: ItemCategory.fwzy, endpoint: "/api/fwzy", file: "public/data/fwzy.json", model: "FwzyItem" }
];

// ============================================
// 7. 导入导出格式
// ============================================

export interface ExportPackage {
  version: string;
  exportedAt: string;
  build: Build;
  metadata?: {
    author?: string;
    description?: string;
    tags?: string[];
  };
}

export interface ImportResult {
  success: boolean;
  build?: Build;
  errors?: string[];
}
