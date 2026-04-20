import { writable, get } from "svelte/store";

export interface ConversationThread {
	id: string;
	title: string;
	route: "goal" | "feature" | "bug" | null;
	messages: ConversationMessage[];
	confirmedIO: ConfirmedIO | null;
	createdAt: string;
	isActive: boolean;
}

export interface ConversationMessage {
	id: string;
	role: "user" | "agent";
	content: string;
	timestamp: string;
	route: "goal" | "feature" | "bug" | null;
	clarificationType: "ask" | "suggest" | "answer" | null;
}

export interface ConfirmedIO {
	id: string;
	threadId: string;
	route: "goal" | "feature" | "bug";
	input: string;
	output: string;
	boundary: string;
	confirmedBy: string;
	confirmedAt: string;
	prototypePath: string | null;
}

function createConversationStore() {
	const { subscribe, set, update } = writable<ConversationThread[]>([]);

	return {
		subscribe,
		addThread: (title: string = "新对话") => {
			const thread: ConversationThread = {
				id: `thread-${Date.now()}`,
				title,
				route: null,
				messages: [],
				confirmedIO: null,
				createdAt: new Date().toISOString(),
				isActive: false,
			};
			update(threads => {
				threads.forEach(t => t.isActive = false);
				return [...threads, thread];
			});
			return thread;
		},
		setActiveThread: (threadId: string) => {
			update(threads => threads.map(t => ({
				...t,
				isActive: t.id === threadId,
			})));
			const threads = get({ subscribe });
			const active = threads.find(t => t.id === threadId);
			if (active && typeof window !== "undefined") {
				window.dispatchEvent(new CustomEvent("thread:switch", {
					detail: { threadId, messages: active.messages }
				}));
			}
		},
		addMessage: (threadId: string, message: Omit<ConversationMessage, "id" | "timestamp">) => {
			update(threads => threads.map(t => {
				if (t.id === threadId) {
					return {
						...t,
						messages: [...t.messages, {
							...message,
							id: `msg-${Date.now()}-${Math.random().toString(36).slice(2)}`,
							timestamp: new Date().toISOString(),
						}]
					};
				}
				return t;
			}));
		},
		setRoute: (threadId: string, route: "goal" | "feature" | "bug") => {
			update(threads => threads.map(t => {
				if (t.id === threadId) return { ...t, route };
				return t;
			}));
		},
		setConfirmedIO: (threadId: string, io: ConfirmedIO) => {
			update(threads => threads.map(t => {
				if (t.id === threadId) return { ...t, confirmedIO: io };
				return t;
			}));
		},
		getActive: () => {
			return get({ subscribe }).find(t => t.isActive) || null;
		},
		removeThread: (threadId: string) => {
			update(threads => threads.filter(t => t.id !== threadId));
		},
	};
}

export const conversationStore = createConversationStore();
