/**
 * Cal4God2 DataCenter 适配层
 * 支持多数据源加载、导入导出、数据查询
 * @version 1.0.0
 */

import { ItemCategory, AnyItem, Build, StatValue, ComboEffect, ComboType } from './types';

// ============================================
// 1. 数据源配置
// ============================================

interface DataSourceConfig {
  category: ItemCategory;
  file: string;
  endpoint: string;
}

const DATA_SOURCES: DataSourceConfig[] = [
  { category: ItemCategory.zk,   file: '/data/zk.json',   endpoint: '/api/zk' },
  { category: ItemCategory.zb,   file: '/data/zb.json',   endpoint: '/api/zb' },
  { category: ItemCategory.tz,   file: '/data/tz.json',   endpoint: '/api/tz' },
  { category: ItemCategory.jb,   file: '/data/jb.json',   endpoint: '/api/jb' },
  { category: ItemCategory.fn,   file: '/data/fn.json',   endpoint: '/api/fn' },
  { category: ItemCategory.hy,   file: '/data/hy.json',   endpoint: '/api/hy' },
  { category: ItemCategory.fw,   file: '/data/fw.json',   endpoint: '/api/fw' },
  { category: ItemCategory.fwzy, file: '/data/fwzy.json', endpoint: '/api/fwzy' },
];

// ============================================
// 2. 适配层接口
// ============================================

export interface DataAdapter {
  // 数据加载
  loadAll(): Promise<Record<ItemCategory, AnyItem[]>>;
  loadCategory(category: ItemCategory): Promise<AnyItem[]>;
  
  // 数据查询
  query(category: ItemCategory, filter?: QueryFilter): Promise<AnyItem[]>;
  getById(category: ItemCategory, id: number): Promise<AnyItem | null>;
  search(category: ItemCategory, keyword: string): Promise<AnyItem[]>;
  
  // 组合计算
  calculateCombos(build: Build): ComboEffect[];
  
  // 导入导出
  exportBuild(build: Build, format: ExportFormat): string;
  importBuild(data: string, format: ExportFormat): Build | null;
}

export interface QueryFilter {
  name?: string;
  property?: keyof StatValue;
  minValue?: number;
  maxValue?: number;
  tags?: string[];
}

export type ExportFormat = 'json' | 'base64' | 'clipboard';

// ============================================
// 3. HTTP 适配器实现
// ============================================

export class HttpDataAdapter implements DataAdapter {
  private baseUrl: string;
  private cache: Map<ItemCategory, AnyItem[]> = new Map();
  private cacheTimeout = 5 * 60 * 1000; // 5分钟缓存
  private cacheTime: Map<ItemCategory, number> = new Map();

  constructor(baseUrl = '') {
    this.baseUrl = baseUrl;
  }

  async loadAll(): Promise<Record<ItemCategory, AnyItem[]>> {
    const results = await Promise.all(
      DATA_SOURCES.map(ds => this.loadCategory(ds.category))
    );
    
    const record: Record<string, AnyItem[]> = {};
    DATA_SOURCES.forEach((ds, i) => {
      record[ds.category] = results[i];
    });
    
    return record as Record<ItemCategory, AnyItem[]>;
  }

  async loadCategory(category: ItemCategory): Promise<AnyItem[]> {
    const now = Date.now();
    const lastLoad = this.cacheTime.get(category) || 0;
    
    // 检查缓存
    if (this.cache.has(category) && (now - lastLoad) < this.cacheTimeout) {
      return this.cache.get(category)!;
    }

    const ds = DATA_SOURCES.find(d => d.category === category);
    if (!ds) return [];

    try {
      const response = await fetch(`${this.baseUrl}${ds.endpoint}`);
      const data = await response.json();
      const items = data.list || [];
      
      this.cache.set(category, items);
      this.cacheTime.set(category, now);
      
      return items;
    } catch (error) {
      console.error(`Failed to load ${category}:`, error);
      return this.cache.get(category) || [];
    }
  }

  async query(category: ItemCategory, filter?: QueryFilter): Promise<AnyItem[]> {
    const items = await this.loadCategory(category);
    
    if (!filter) return items;
    
    return items.filter(item => {
      if (filter.name && !item.name.includes(filter.name)) return false;
      if (filter.property && filter.minValue !== undefined) {
        const val = item.sx[filter.property] || 0;
        if (val < filter.minValue) return false;
      }
      if (filter.property && filter.maxValue !== undefined) {
        const val = item.sx[filter.property] || 0;
        if (val > filter.maxValue) return false;
      }
      return true;
    });
  }

  async getById(category: ItemCategory, id: number): Promise<AnyItem | null> {
    const items = await this.loadCategory(category);
    return items.find(item => item.id === id) || null;
  }

  async search(category: ItemCategory, keyword: string): Promise<AnyItem[]> {
    const items = await this.loadCategory(category);
    const lower = keyword.toLowerCase();
    return items.filter(item => 
      item.name.toLowerCase().includes(lower)
    );
  }

  calculateCombos(build: Build): ComboEffect[] {
    const combos: ComboEffect[] = [];
    
    // 计算套装羁绊
    combos.push(...this.calculateTzCombos(build));
    
    // 计算羁绊效果
    combos.push(...this.calculateJbCombos(build));
    
    // 计算符文之语
    combos.push(...this.calculateFwzyCombos(build));
    
    return combos;
  }

  private calculateTzCombos(build: Build): ComboEffect[] {
    // 套装计算逻辑
    const equippedTypes = build.zbSelection.map(z => z.type);
    const typeCounts = equippedTypes.reduce((acc, type) => {
      acc[type] = (acc[type] || 0) + 1;
      return acc;
    }, {} as Record<string, number>);
    
    return []; // TODO: 实现套装激活判断
  }

  private calculateJbCombos(build: Build): ComboEffect[] {
    // 羁绊计算逻辑
    return [];
  }

  private calculateFwzyCombos(build: Build): ComboEffect[] {
    // 符文之语计算逻辑
    return [];
  }

  // ============================================
  // 4. 导入导出实现
  // ============================================

  exportBuild(build: Build, format: ExportFormat): string {
    const package_ = {
      version: '1.0.0',
      exportedAt: new Date().toISOString(),
      build: this.sanitizeBuild(build)
    };

    switch (format) {
      case 'json':
        return JSON.stringify(package_, null, 2);
      
      case 'base64':
        const json = JSON.stringify(package_);
        const compressed = this.compress(json);
        return btoa(compressed);
      
      case 'clipboard':
        return JSON.stringify(package_);
      
      default:
        return JSON.stringify(package_);
    }
  }

  importBuild(data: string, format: ExportFormat): Build | null {
    try {
      let package_: any;
      
      switch (format) {
        case 'json':
          package_ = JSON.parse(data);
          break;
        
        case 'base64':
          const decompressed = atob(data);
          package_ = JSON.parse(this.decompress(decompressed));
          break;
        
        case 'clipboard':
          package_ = JSON.parse(data);
          break;
        
        default:
          return null;
      }
      
      if (package_.version && package_.build) {
        return this.validateBuild(package_.build);
      }
      
      return null;
    } catch (error) {
      console.error('Import failed:', error);
      return null;
    }
  }

  private sanitizeBuild(build: Build): Build {
    // 清理 Build 对象，移除不可序列化的部分
    return {
      ...build,
      id: build.id || crypto.randomUUID(),
      name: build.name || '未命名方案',
      createdAt: build.createdAt || new Date().toISOString(),
      updatedAt: new Date().toISOString()
    };
  }

  private validateBuild(obj: any): Build | null {
    // 基本验证
    if (!obj || typeof obj !== 'object') return null;
    
    return {
      id: obj.id || crypto.randomUUID(),
      name: obj.name || '导入方案',
      createdAt: obj.createdAt || new Date().toISOString(),
      updatedAt: obj.updatedAt || new Date().toISOString(),
      zkSelection: Array.isArray(obj.zkSelection) ? obj.zkSelection : [],
      zbSelection: Array.isArray(obj.zbSelection) ? obj.zbSelection : [],
      fnSelection: Array.isArray(obj.fnSelection) ? obj.fnSelection : [],
      jbSelection: Array.isArray(obj.jbSelection) ? obj.jbSelection : [],
      hySelection: Array.isArray(obj.hySelection) ? obj.hySelection : [],
      fwSelection: Array.isArray(obj.fwSelection) ? obj.fwSelection : [],
      fwzySelection: Array.isArray(obj.fwzySelection) ? obj.fwzySelection : [],
      totalStats: obj.totalStats || {},
      activeCombos: Array.isArray(obj.activeCombos) ? obj.activeCombos : []
    };
  }

  private compress(str: string): string {
    // 简单的压缩实现（实际项目可用 pako/lz-string）
    return encodeURIComponent(str);
  }

  private decompress(str: string): string {
    return decodeURIComponent(str);
  }
}

// ============================================
// 5. LocalStorage 适配器 (离线模式)
// ============================================

export class LocalStorageAdapter implements DataAdapter {
  private prefix = 'cal4god2_';
  private httpAdapter: HttpDataAdapter;

  constructor() {
    this.httpAdapter = new HttpDataAdapter();
  }

  async loadAll(): Promise<Record<ItemCategory, AnyItem[]>> {
    return this.httpAdapter.loadAll();
  }

  async loadCategory(category: ItemCategory): Promise<AnyItem[]> {
    const key = `${this.prefix}${category}`;
    const cached = localStorage.getItem(key);
    
    if (cached) {
      return JSON.parse(cached);
    }
    
    const items = await this.httpAdapter.loadCategory(category);
    localStorage.setItem(key, JSON.stringify(items));
    
    return items;
  }

  async query(category: ItemCategory, filter?: QueryFilter): Promise<AnyItem[]> {
    return this.httpAdapter.query(category, filter);
  }

  async getById(category: ItemCategory, id: number): Promise<AnyItem | null> {
    return this.httpAdapter.getById(category, id);
  }

  async search(category: ItemCategory, keyword: string): Promise<AnyItem[]> {
    return this.httpAdapter.search(category, keyword);
  }

  calculateCombos(build: Build): ComboEffect[] {
    return this.httpAdapter.calculateCombos(build);
  }

  exportBuild(build: Build, format: ExportFormat): string {
    return this.httpAdapter.exportBuild(build, format);
  }

  importBuild(data: string, format: ExportFormat): Build | null {
    return this.httpAdapter.importBuild(data, format);
  }

  // 本地存储方法
  saveBuild(build: Build): void {
    const builds = this.getSavedBuilds();
    const existing = builds.findIndex(b => b.id === build.id);
    
    if (existing >= 0) {
      builds[existing] = build;
    } else {
      builds.push(build);
    }
    
    localStorage.setItem(`${this.prefix}builds`, JSON.stringify(builds));
  }

  getSavedBuilds(): Build[] {
    const data = localStorage.getItem(`${this.prefix}builds`);
    return data ? JSON.parse(data) : [];
  }

  deleteBuild(id: string): void {
    const builds = this.getSavedBuilds().filter(b => b.id !== id);
    localStorage.setItem(`${this.prefix}builds`, JSON.stringify(builds));
  }
}

// ============================================
// 6. 工具函数
// ============================================

export function mergeStats(...stats: StatValue[]): StatValue {
  const result: StatValue = {};
  
  for (const stat of stats) {
    for (const [key, value] of Object.entries(stat)) {
      if (value !== undefined && value !== 0) {
        result[key] = (result[key] || 0) + value;
      }
    }
  }
  
  return result;
}

export function calculateTotalStats(build: Build): StatValue {
  const allStats: StatValue[] = [];
  
  // 累加所有已选物品的词条
  build.zkSelection.forEach(item => allStats.push(item.sx));
  build.zbSelection.forEach(item => allStats.push(item.sx));
  build.fnSelection.forEach(item => allStats.push(item.sx));
  build.jbSelection.forEach(item => allStats.push(item.sx));
  build.hySelection.forEach(item => allStats.push(item.sx));
  build.fwSelection.forEach(item => allStats.push(item.sx));
  build.fwzySelection.forEach(item => allStats.push(item.sx));
  
  // 累加激活的组合效果
  build.activeCombos
    .filter(c => c.isActive)
    .forEach(c => allStats.push(c.stats));
  
  return mergeStats(...allStats);
}

// ============================================
// 7. 默认导出
// ============================================

export const defaultAdapter = new HttpDataAdapter();
export const localAdapter = new LocalStorageAdapter();
