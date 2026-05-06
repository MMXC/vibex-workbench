/**
 * Intent → UI Spec → Prototype Gate（从当前 spec YAML 解析 prototype.intent / prototype.ui_spec）
 * 与 pkg/designkit/gate.go 规则保持一致。
 */
import { parse as parseYaml } from 'yaml';

export type UiWorkflowStage = 'intent_captured' | 'ui_spec_drafted' | 'gate_pending' | 'gate_passed';

export type GateFailureCode =
	| 'intent_incomplete'
	| 'ui_spec_incomplete'
	| 'acceptance_missing'
	| 'confirm_required';

export type GateCheck = {
	id: string;
	label: string;
	passed: boolean;
	detail?: string;
};

export type UiWorkflowGateModel = {
	stage: UiWorkflowStage;
	checks: GateCheck[];
	failedCodes: GateFailureCode[];
	canCommitPrototype: boolean;
	nextAction: string;
};

function asRecord(value: unknown): Record<string, unknown> | null {
	return value && typeof value === 'object' && !Array.isArray(value)
		? (value as Record<string, unknown>)
		: null;
}

function nonEmptyStr(v: unknown): boolean {
	return typeof v === 'string' && v.trim().length > 0;
}

function nonEmptyList(v: unknown): boolean {
	if (v == null) return false;
	if (Array.isArray(v)) return v.some(nonEmptyStr);
	return false;
}

const REQUIRED_STATES = ['loading', 'empty', 'error', 'normal'];

function statesSatisfied(raw: unknown): boolean {
	if (raw == null) return false;
	const names = new Set<string>();
	if (Array.isArray(raw)) {
		for (const x of raw) {
			if (nonEmptyStr(x)) names.add(String(x).trim().toLowerCase());
		}
	} else if (typeof raw === 'object') {
		for (const k of Object.keys(raw as Record<string, unknown>)) {
			names.add(k.trim().toLowerCase());
		}
	} else if (nonEmptyStr(raw)) {
		const s = String(raw).toLowerCase();
		for (const st of REQUIRED_STATES) {
			if (s.includes(st)) names.add(st);
		}
	}
	return REQUIRED_STATES.every(s => names.has(s));
}

function responsiveSatisfied(raw: unknown): boolean {
	const o = asRecord(raw);
	if (!o) return nonEmptyStr(raw);
	const d = nonEmptyStr(o.desktop);
	const m = nonEmptyStr(o.mobile);
	const sum = nonEmptyStr(o.summary);
	return (d && m) || sum;
}

/** 若解析失败，视为 gate 未通过（提示用户修复 YAML） */
export function evaluatePrototypeGate(specYaml: string): UiWorkflowGateModel {
	const checks: GateCheck[] = [];
	let doc: Record<string, unknown> | null = null;
	try {
		doc = parseYaml(specYaml) as Record<string, unknown>;
	} catch {
		checks.push({
			id: 'yaml_parse',
			label: 'Spec YAML 可解析',
			passed: false,
			detail: '当前 spec 内容不是合法 YAML，无法解析 prototype 门禁字段。',
		});
		return buildGateModel(checks);
	}

	const prototype = asRecord(doc?.prototype);
	const intent = asRecord(prototype?.intent);
	const uiSpec = asRecord(prototype?.ui_spec);

	const goal = intent?.business_goal ?? intent?.goal;

	checks.push({
		id: 'intent_target_user',
		label: 'Intent：目标用户（prototype.intent.target_user）',
		passed: nonEmptyStr(intent?.target_user),
	});
	checks.push({
		id: 'intent_goal',
		label: 'Intent：业务目标（prototype.intent.business_goal 或 goal）',
		passed: nonEmptyStr(goal),
	});
	checks.push({
		id: 'intent_primary_action',
		label: 'Intent：核心动作（prototype.intent.primary_action）',
		passed: nonEmptyStr(intent?.primary_action),
	});
	checks.push({
		id: 'intent_success',
		label: 'Intent：成功标准（prototype.intent.success_criteria）',
		passed: nonEmptyStr(intent?.success_criteria) || nonEmptyList(intent?.success_criteria),
	});

	const sections = uiSpec?.sections;
	const sectionOk =
		Array.isArray(sections) &&
		sections.length > 0 &&
		sections.every(s => {
			const sec = asRecord(s);
			return sec && nonEmptyStr(sec.name) && nonEmptyStr(sec.component_type ?? sec['component-type']);
		});

	checks.push({
		id: 'ui_page_purpose',
		label: 'UI Spec：页面目的（prototype.ui_spec.page_purpose）',
		passed: nonEmptyStr(uiSpec?.page_purpose),
	});
	checks.push({
		id: 'ui_sections',
		label: 'UI Spec：模块列表（prototype.ui_spec.sections，≥1 且含 name、component_type）',
		passed: !!sectionOk,
	});
	checks.push({
		id: 'ui_states',
		label: 'UI Spec：状态矩阵含 loading/empty/error/normal（prototype.ui_spec.states）',
		passed: statesSatisfied(uiSpec?.states),
	});
	checks.push({
		id: 'ui_responsive',
		label: 'UI Spec：响应式（prototype.ui_spec.responsive.desktop+mobile 或 summary）',
		passed: responsiveSatisfied(uiSpec?.responsive),
	});
	checks.push({
		id: 'ui_acceptance',
		label: 'UI Spec：验收要点（prototype.ui_spec.acceptance）',
		passed: nonEmptyStr(uiSpec?.acceptance) || nonEmptyList(uiSpec?.acceptance),
	});

	return buildGateModel(checks);
}

function buildGateModel(checks: GateCheck[]): UiWorkflowGateModel {
	const failed = checks.filter(c => !c.passed);
	const failedCodes = new Set<GateFailureCode>();

	for (const c of failed) {
		if (c.id.startsWith('intent_')) failedCodes.add('intent_incomplete');
		else if (c.id === 'ui_acceptance') failedCodes.add('acceptance_missing');
		else if (c.id.startsWith('ui_')) failedCodes.add('ui_spec_incomplete');
	}

	const uiComplete = failed.length === 0;
	const intentFailed = failed.some(f => f.id.startsWith('intent_'));
	const parseFailed = failed.some(f => f.id === 'yaml_parse');

	let stage: UiWorkflowStage;
	if (uiComplete) {
		stage = 'gate_passed';
	} else if (parseFailed || intentFailed) {
		stage = 'gate_pending';
	} else {
		// intent 检查已通过，UI Spec / 验收仍缺
		stage = 'ui_spec_drafted';
	}

	const nextAction = buildNextAction(failed);

	return {
		stage,
		checks,
		failedCodes: [...failedCodes],
		canCommitPrototype: failed.length === 0,
		nextAction,
	};
}

function buildNextAction(failed: GateCheck[]): string {
	if (failed.length === 0) return '已通过 Prototype Gate，可进行物料提取或初始化（仍需 confirm）。';
	const lines = failed.map(f => `- ${f.label}${f.detail ? `：${f.detail}` : ''}`);
	return [
		'请在当前 spec 的 `prototype:` 下补齐以下 YAML 块（可与 Agent 协作），然后保存/刷新槽位：',
		'```yaml',
		'prototype:',
		'  intent:',
		'    target_user: "<谁在用>"',
		'    business_goal: "<解决什么问题>"',
		'    primary_action: "<用户最关键的操作>"',
		'    success_criteria: "<怎样算验收通过>"',
		'  ui_spec:',
		'    page_purpose: "<页面目的>"',
		'    sections:',
		'      - name: "<模块名>"',
		'        component_type: "<卡片|表格|…>"',
		'        priority: high',
		'    states: [loading, empty, error, normal]',
		'    responsive:',
		'      desktop: "<桌面策略摘要>"',
		'      mobile: "<移动策略摘要>"',
		'    acceptance:',
		'      - "<验收项 1>"',
		'```',
		'未通过项：',
		...lines,
	].join('\n');
}
