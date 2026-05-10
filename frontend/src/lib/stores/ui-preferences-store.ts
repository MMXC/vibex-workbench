import { writable } from 'svelte/store';
import { browser } from '$app/environment';

type UiPreferences = {
	canvasAutoUiEnabled: boolean;
};

const STORAGE_KEY = 'vibex-ui-preferences';

function loadInitial(): UiPreferences {
	if (!browser) return { canvasAutoUiEnabled: true };
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) return { canvasAutoUiEnabled: true };
		const parsed = JSON.parse(raw) as Partial<UiPreferences>;
		return {
			canvasAutoUiEnabled: parsed.canvasAutoUiEnabled !== false,
		};
	} catch {
		return { canvasAutoUiEnabled: true };
	}
}

function createUiPreferencesStore() {
	const { subscribe, update } = writable<UiPreferences>(loadInitial());

	return {
		subscribe,
		toggleCanvasAutoUi() {
			update(state => {
				const next = { ...state, canvasAutoUiEnabled: !state.canvasAutoUiEnabled };
				if (browser) localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
				return next;
			});
		},
	};
}

export const uiPreferencesStore = createUiPreferencesStore();
