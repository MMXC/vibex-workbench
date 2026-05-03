<script lang="ts">
	import { onMount } from 'svelte';
	import {
		hasNativeWindowHost,
		windowGetPosition,
		windowGetSize,
		windowIsMaximized,
		windowResizeTo,
	} from '$lib/wails-runtime';

	type ResizeEdge = 'n' | 'e' | 's' | 'w' | 'ne' | 'nw' | 'se' | 'sw';

	const minWidth = 800;
	const minHeight = 600;
	let enabled = $state(false);

	onMount(() => {
		enabled = hasNativeWindowHost();
	});

	async function beginResize(edge: ResizeEdge, event: PointerEvent) {
		if (!enabled || event.button !== 0) return;
		if (await windowIsMaximized()) return;

		const startSize = await windowGetSize();
		const startPosition = await windowGetPosition();
		if (!startSize || !startPosition) return;
		const size = startSize;
		const position = startPosition;

		event.preventDefault();
		event.stopPropagation();

		const startX = event.screenX;
		const startY = event.screenY;

		function applyResize(moveEvent: PointerEvent) {
			moveEvent.preventDefault();
			const dx = moveEvent.screenX - startX;
			const dy = moveEvent.screenY - startY;

			let nextX = position.x;
			let nextY = position.y;
			let nextW = size.w;
			let nextH = size.h;

			if (edge.includes('e')) {
				nextW = Math.max(minWidth, size.w + dx);
			}
			if (edge.includes('s')) {
				nextH = Math.max(minHeight, size.h + dy);
			}
			if (edge.includes('w')) {
				nextW = Math.max(minWidth, size.w - dx);
				nextX = position.x + (size.w - nextW);
			}
			if (edge.includes('n')) {
				nextH = Math.max(minHeight, size.h - dy);
				nextY = position.y + (size.h - nextH);
			}

			void windowResizeTo(nextX, nextY, nextW, nextH);
		}

		function endResize() {
			window.removeEventListener('pointermove', applyResize);
			window.removeEventListener('pointerup', endResize);
			window.removeEventListener('pointercancel', endResize);
		}

		window.addEventListener('pointermove', applyResize);
		window.addEventListener('pointerup', endResize, { once: true });
		window.addEventListener('pointercancel', endResize, { once: true });
	}
</script>

{#if enabled}
	<div class="resize-frame" aria-hidden="true">
		<button type="button" class="resize-hit resize-hit-n" aria-label="向上调整窗口大小" onpointerdown={(event) => beginResize('n', event)} tabindex="-1"></button>
		<button type="button" class="resize-hit resize-hit-e" aria-label="向右调整窗口大小" onpointerdown={(event) => beginResize('e', event)} tabindex="-1"></button>
		<button type="button" class="resize-hit resize-hit-s" aria-label="向下调整窗口大小" onpointerdown={(event) => beginResize('s', event)} tabindex="-1"></button>
		<button type="button" class="resize-hit resize-hit-w" aria-label="向左调整窗口大小" onpointerdown={(event) => beginResize('w', event)} tabindex="-1"></button>
		<button type="button" class="resize-hit resize-hit-ne" aria-label="向右上调整窗口大小" onpointerdown={(event) => beginResize('ne', event)} tabindex="-1"></button>
		<button type="button" class="resize-hit resize-hit-nw" aria-label="向左上调整窗口大小" onpointerdown={(event) => beginResize('nw', event)} tabindex="-1"></button>
		<button type="button" class="resize-hit resize-hit-se" aria-label="向右下调整窗口大小" onpointerdown={(event) => beginResize('se', event)} tabindex="-1"></button>
		<button type="button" class="resize-hit resize-hit-sw" aria-label="向左下调整窗口大小" onpointerdown={(event) => beginResize('sw', event)} tabindex="-1"></button>
	</div>
{/if}

<style>
	.resize-frame {
		position: fixed;
		inset: 0;
		z-index: 10000;
		pointer-events: none;
		--wails-draggable: no-drag;
	}

	.resize-hit {
		position: absolute;
		z-index: 1;
		padding: 0;
		border: 0;
		background: transparent;
		pointer-events: auto;
		--wails-draggable: no-drag;
	}

	.resize-hit-n,
	.resize-hit-s {
		left: 8px;
		right: 8px;
		height: 6px;
		cursor: ns-resize;
	}

	.resize-hit-e,
	.resize-hit-w {
		top: 8px;
		bottom: 8px;
		width: 6px;
		cursor: ew-resize;
	}

	.resize-hit-n {
		top: 0;
	}

	.resize-hit-e {
		right: 0;
	}

	.resize-hit-s {
		bottom: 0;
	}

	.resize-hit-w {
		left: 0;
	}

	.resize-hit-ne,
	.resize-hit-nw,
	.resize-hit-se,
	.resize-hit-sw {
		width: 12px;
		height: 12px;
	}

	.resize-hit-ne {
		top: 0;
		right: 0;
		cursor: nesw-resize;
	}

	.resize-hit-nw {
		top: 0;
		left: 0;
		cursor: nwse-resize;
	}

	.resize-hit-se {
		right: 0;
		bottom: 0;
		cursor: nwse-resize;
	}

	.resize-hit-sw {
		left: 0;
		bottom: 0;
		cursor: nesw-resize;
	}
</style>
