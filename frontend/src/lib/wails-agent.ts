/**
 * Wails Agent — spawn Go agent subprocess via Wails binding
 *
 * Production (Wails): window.go.main.App.RunAgent spawns vibex-agent subprocess
 * Development: not available (agent requires Go binary), falls back to error
 */

import { eventsOn, isWails } from './wails-runtime';

// ── Types ─────────────────────────────────────────────────────

export interface AgentGoalRequest {
	goal: string;
	workspaceRoot: string;
}

export interface AgentEvent {
	type: 'tool_call' | 'tool_output' | 'message' | 'done' | 'error';
	data: any;
}

// ── State ─────────────────────────────────────────────────────

let currentPid: number | null = null;
let killFn: (() => void) | null = null;

// ── RunAgent ──────────────────────────────────────────────────

/**
 * 通过 Wails binding spawn agent subprocess。
 *
 * agent stdout 每行 JSON 被解析为 AgentEvent，通过 onEvent 回调传给前端。
 * agent 执行完毕后 onDone 被调用。
 *
 * @param goal          用户输入的 goal 描述
 * @param workspaceRoot 工作区根路径
 * @param onEvent       每条 agent event 的回调（tool_call / tool_output / message）
 * @param onDone        agent 执行完毕后的回调（最终 answer）
 * @param onError       错误回调
 * @returns handle with pid and kill()
 */
export function runAgent(
	goal: string,
	workspaceRoot: string,
	onEvent: (event: AgentEvent) => void,
	onDone: (answer: string) => void,
	onError: (err: string) => void
): { pid: number; kill: () => void } {
	if (!isWails()) {
		onError('Agent 模式需要 Wails 桌面环境（浏览器开发模式不支持）');
		return { pid: 0, kill: () => {} };
	}
	const app = (window as any).go?.main?.App;
	if (!app || typeof app.RunAgent !== 'function' || typeof app.KillAgent !== 'function') {
		onError('Wails binding missing: App.RunAgent/KillAgent');
		return { pid: 0, kill: () => {} };
	}

	// 订阅 agent event（由 main.go RunAgent 转发）
	const offStdout = eventsOn('agent:stdout', (line: string) => {
		try {
			const event: AgentEvent = JSON.parse(line as string);
			onEvent(event);
		} catch {
			// 非 JSON 行，可能是普通输出，直接当 message 处理
			onEvent({ type: 'message', data: { content: line } });
		}
	});

	const offDone = eventsOn('agent:done', (_payload: any) => {
		currentPid = null;
		offStdout();
		offDone();
		offError();
	});

	const offError = eventsOn('agent:error', (err: string) => {
		onError(err as string);
		currentPid = null;
		offStdout();
		offDone();
		offError();
	});

	// 调用 Wails binding spawn agent
	const req: AgentGoalRequest = { goal, workspaceRoot };
	app.RunAgent(undefined, JSON.stringify(req))
		.then((result: any) => {
			currentPid = result.pid;
		})
		.catch((err: any) => {
			onError(String(err));
			offStdout();
			offDone();
			offError();
		});

	const kill: () => void = () => {
		if (currentPid !== null) {
			app.KillAgent(undefined, currentPid);
			currentPid = null;
		}
	};

	killFn = kill;
	return { pid: currentPid ?? 0, kill };
}

/**
 * 终止当前正在运行的 agent subprocess。
 */
export function killAgent(): void {
	if (killFn) {
		killFn();
		killFn = null;
	}
}
