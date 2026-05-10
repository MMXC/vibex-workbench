import { writable } from 'svelte/store';
import type { Message } from '$lib/stores/thread-store';

export type AiBlockKind = 'command' | 'json' | 'code' | 'section' | 'item';

export interface AiBlock {
	id: string;
	threadId: string;
	messageId: string;
	kind: AiBlockKind;
	title: string;
	content: string;
	command?: string;
}

export type CanvasSkillAction = {
	id: string;
	label: string;
	type: 'prefill_composer' | 'request_api' | 'open_slot' | 'pin_canvas' | 'cancel';
	confirm?: string;
	variant?: 'primary' | 'secondary' | 'danger';
	payload?: Record<string, unknown>;
};

export type CanvasSkillPayload = {
	template: string;
	title: string;
	summary?: string;
	blocks?: Array<{
		type: string;
		title?: string;
		items?: string[];
		data?: Record<string, unknown>;
		content?: string;
	}>;
	actions?: CanvasSkillAction[];
};

type AiBlocksState = {
	threadId: string | null;
	blocks: AiBlock[];
	skillPayload: CanvasSkillPayload | null;
};

function looksLikeCanvasSkillPayload(v: unknown): v is CanvasSkillPayload {
	if (!v || typeof v !== 'object') return false;
	const o = v as Record<string, unknown>;
	return typeof o.template === 'string' && typeof o.title === 'string';
}

function extractSkillPayloadFromMessage(message: Message): CanvasSkillPayload | null {
	if (message.role !== 'assistant') return null;
	const text = message.content ?? '';
	const fencePattern = /```json\s*([\s\S]*?)```/gi;
	let fm: RegExpExecArray | null = null;
	while ((fm = fencePattern.exec(text))) {
		const raw = fm[1]?.trim();
		if (!raw) continue;
		try {
			const parsed = JSON.parse(raw);
			if (looksLikeCanvasSkillPayload(parsed)) return parsed;
		} catch {
			// ignore invalid json fence
		}
	}
	// fallback: entire message is json
	const whole = text.trim();
	if (whole.startsWith('{') && whole.endsWith('}')) {
		try {
			const parsed = JSON.parse(whole);
			if (looksLikeCanvasSkillPayload(parsed)) return parsed;
		} catch {
			// ignore invalid whole json
		}
	}
	return null;
}

function extractBlocksFromMessage(threadId: string, message: Message): AiBlock[] {
	if (message.role !== 'assistant') return [];
	const text = message.content ?? '';
	const blocks: AiBlock[] = [];

	const commandLines = text
		.split('\n')
		.map(line => line.trim())
		.filter(line => line.startsWith('/'));
	for (const cmd of commandLines.slice(0, 4)) {
		blocks.push({
			id: `${message.id}:cmd:${blocks.length}`,
			threadId,
			messageId: message.id,
			kind: 'command',
			title: `命令 · ${cmd.split(/\s+/)[0]}`,
			content: cmd,
			command: cmd,
		});
	}

	const codeFence = /```([a-zA-Z0-9_-]*)\n([\s\S]*?)```/g;
	let m: RegExpExecArray | null = null;
	while ((m = codeFence.exec(text)) && blocks.length < 12) {
		const lang = (m[1] || '').toLowerCase();
		const body = (m[2] || '').trim();
		if (!body) continue;
		const kind: AiBlockKind = lang === 'json' ? 'json' : 'code';
		blocks.push({
			id: `${message.id}:fence:${blocks.length}`,
			threadId,
			messageId: message.id,
			kind,
			title: kind === 'json' ? 'JSON 结构块' : `代码块${lang ? ` · ${lang}` : ''}`,
			content: body,
		});
	}

	const headings = text
		.split('\n')
		.map(line => line.trim())
		.filter(line => /^#{1,3}\s+/.test(line))
		.slice(0, 4);
	for (const heading of headings) {
		blocks.push({
			id: `${message.id}:sec:${blocks.length}`,
			threadId,
			messageId: message.id,
			kind: 'section',
			title: '章节',
			content: heading.replace(/^#{1,3}\s+/, ''),
		});
	}

	const items = text
		.split('\n')
		.map(line => line.trim())
		.filter(line => /^([-*]|\d+\.)\s+/.test(line))
		.slice(0, 6);
	for (const item of items) {
		blocks.push({
			id: `${message.id}:item:${blocks.length}`,
			threadId,
			messageId: message.id,
			kind: 'item',
			title: '要点',
			content: item.replace(/^([-*]|\d+\.)\s+/, ''),
		});
	}

	return blocks.slice(0, 12);
}

function createAiBlocksStore() {
	const { subscribe, set } = writable<AiBlocksState>({
		threadId: null,
		blocks: [],
		skillPayload: null,
	});

	return {
		subscribe,
		ingestThread(threadId: string | null, messages: Message[]) {
			if (!threadId) {
				set({ threadId: null, blocks: [], skillPayload: null });
				return;
			}
			const assistants = messages.filter(m => m.role === 'assistant').slice(-4);
			const blocks = assistants.flatMap(msg => extractBlocksFromMessage(threadId, msg)).slice(-18);
			let skillPayload: CanvasSkillPayload | null = null;
			for (let i = assistants.length - 1; i >= 0; i--) {
				const found = extractSkillPayloadFromMessage(assistants[i]);
				if (found) {
					skillPayload = found;
					break;
				}
			}
			set({ threadId, blocks, skillPayload });
		},
	};
}

export const aiBlocksStore = createAiBlocksStore();
