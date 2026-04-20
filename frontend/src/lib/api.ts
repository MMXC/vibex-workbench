// ============================================================
// ⚠️  此文件由 spec-to-sveltekit 自动生成
//     来自: specs/feature
//     生成时间: 2026-04-20
//     模式: backend
//
// ⚠️  不要直接编辑此文件
//     修改 specs/ 目录下的 YAML 文件后重新运行 make generate-frontend
// ============================================================

import type {
  FileDiff,
  FileDiffCreate,
  FileDiffUpdate,
  GenerationJob,
  GenerationJobCreate,
  GenerationJobUpdate,
  ImpactReport,
  ImpactReportCreate,
  ImpactReportUpdate,
  SpecAST,
  SpecASTCreate,
  SpecASTUpdate,
  SpecFile,
  SpecFileCreate,
  SpecFileUpdate,
  SpecLocation,
  SpecLocationCreate,
  SpecLocationUpdate,
  SpecNode,
  SpecNodeCreate,
  SpecNodeUpdate,
  Viewport,
  ViewportCreate,
  ViewportUpdate,
  Violation,
  ViolationCreate,
  ViolationUpdate,
  DependencyGraph,
  Change,
} from './types';

const API_BASE = '/api';

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(API_BASE + path, {
    headers: { 'Content-Type': 'application/json', ...options.headers },
    ...options,
  });
  if (!res.ok) throw new Error(`API ${path} failed: ${res.status}`);
  return res.json();
}

// ── Generated API ──────────────────────────────────────────────

export async function generateFromGraph(graph: DependencyGraph): Promise<string> {
  //   // MermaidGeneratorService.generateFromGraph -> string
  return request<string>(`/api/mermaidgeneratorservice/generatefromgraph`);
}

export async function generateFromSpecs(specs: SpecAST[]): Promise<string> {
  //   // MermaidGeneratorService.generateFromSpecs -> string
  return request<string>(`/api/mermaidgeneratorservice/generatefromspecs`);
}

export async function autoLayout(nodes: SpecNode[]): Promise<SpecNode[]> {
  //   // CanvasLayoutService.autoLayout -> SpecNode[]
  return request<SpecNode[]>(`/api/canvaslayoutservice/autolayout`);
}

export async function fitToScreen(nodes: SpecNode[], viewport: Viewport): Promise<Viewport> {
  //   // CanvasLayoutService.fitToScreen -> Viewport
  return request<Viewport>(`/api/canvaslayoutservice/fittoscreen`);
}

export async function runGenerate(): Promise<GenerationJob> {
  //   // GeneratorService.runGenerate -> GenerationJob
  return request<GenerationJob>(`/api/generatorservice/rungenerate`);
}

export async function getDiff(specId: string): Promise<FileDiff[]> {
  //   // GeneratorService.getDiff -> FileDiff[]
  return request<FileDiff[]>(`/api/generatorservice/getdiff`);
}

export async function routeChange(change: Change): Promise<SpecLocation[]> {
  //   // RouterService.routeChange -> SpecLocation[]
  return request<SpecLocation[]>(`/api/routerservice/routechange`);
}

export async function analyzeImpact(specId: string): Promise<ImpactReport> {
  //   // ImpactService.analyzeImpact -> ImpactReport
  return request<ImpactReport>(`/api/impactservice/analyzeimpact`);
}

export async function toggleLeftSidebar(): Promise<void> {
  //   // LayoutService.toggleLeftSidebar -> void
  return request<void>(`/api/layoutservice/toggleleftsidebar`);
}

export async function toggleRightSidebar(): Promise<void> {
  //   // LayoutService.toggleRightSidebar -> void
  return request<void>(`/api/layoutservice/togglerightsidebar`);
}

export async function setActivePanel(panel: string): Promise<void> {
  //   // LayoutService.setActivePanel -> void
  return request<void>(`/api/layoutservice/setactivepanel`);
}

export async function showMessage(msg: string, type: string): Promise<void> {
  //   // StatusService.showMessage -> void
  return request<void>(`/api/statusservice/showmessage`);
}

export async function showGenerationStatus(status: string): Promise<void> {
  //   // StatusService.showGenerationStatus -> void
  return request<void>(`/api/statusservice/showgenerationstatus`);
}

export async function parseYAML(content: string): Promise<SpecAST> {
  //   // SpecParserService.parseYAML -> SpecAST
  return request<SpecAST>(`/api/specparserservice/parseyaml`);
}

export async function serializeYAML(ast: SpecAST): Promise<string> {
  //   // SpecParserService.serializeYAML -> string
  return request<string>(`/api/specparserservice/serializeyaml`);
}

export async function validateParentChain(spec: SpecAST, allSpecs: SpecAST[]): Promise<Violation[]> {
  //   // ValidationService.validateParentChain -> Violation[]
  return request<Violation[]>(`/api/validationservice/validateparentchain`);
}

export async function validateLevelConsistency(spec: SpecAST): Promise<Violation[]> {
  //   // ValidationService.validateLevelConsistency -> Violation[]
  return request<Violation[]>(`/api/validationservice/validatelevelconsistency`);
}

export async function readSpec(path: string): Promise<SpecFile> {
  //   // FileService.readSpec -> SpecFile
  return request<SpecFile>(`/api/fileservice/readspec`);
}

export async function writeSpec(path: string, content: string): Promise<void> {
  //   // FileService.writeSpec -> void
  return request<void>(`/api/fileservice/writespec`);
}

export async function listSpecs(): Promise<SpecFile[]> {
  //   // FileService.listSpecs -> SpecFile[]
  return request<SpecFile[]>(`/api/fileservice/listspecs`);
}

