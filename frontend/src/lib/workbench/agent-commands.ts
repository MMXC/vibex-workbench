import { agentApiUrl } from '$lib/runtime/agent-transport';

export type AgentCommand = {
	command: string;
	label_zh: string;
	source?: string;
};

const cache = new Map<string, { at: number; commands: AgentCommand[] }>();
const TTL_MS = 60_000;

export async function fetchAgentCommands(workspaceRoot: string | null | undefined): Promise<AgentCommand[]> {
	const ws = workspaceRoot?.trim();
	if (!ws) return [];
	const hit = cache.get(ws);
	if (hit && Date.now() - hit.at < TTL_MS) return hit.commands;
	const url = `${agentApiUrl('/api/workspace/agent-commands')}?workspaceRoot=${encodeURIComponent(ws)}`;
	try {
		const r = await fetch(url);
		if (!r.ok) return [];
		const j = (await r.json()) as { commands?: AgentCommand[] };
		const commands = Array.isArray(j.commands) ? j.commands : [];
		cache.set(ws, { at: Date.now(), commands });
		return commands;
	} catch {
		return [];
	}
}
