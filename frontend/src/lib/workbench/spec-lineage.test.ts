import { describe, expect, it } from 'vitest';
import { collectChildPathsForParent, resolveSpecRefToPath } from './spec-lineage';

describe('spec-lineage', () => {
	it('collectChildPathsForParent matches parent name and dedupes', () => {
		const specs = [
			{ path: '.vibex/specs/L4-feature/A.yaml', parent: 'MOD-x' },
			{ path: '.vibex/specs/L4-feature/B.yaml', parent: 'MOD-x' },
			{ path: '.vibex/specs/L4-feature/C.yaml', parent: 'OTHER' },
		];
		expect(collectChildPathsForParent(specs, 'MOD-x')).toEqual([
			'.vibex/specs/L4-feature/A.yaml',
			'.vibex/specs/L4-feature/B.yaml',
		]);
	});

	it('collectChildPathsForParent excludes self path', () => {
		const specs = [{ path: '.vibex/specs/L2-skeleton/p.yaml', parent: 'goal' }];
		expect(
			collectChildPathsForParent(specs, 'goal', {
				excludeSelfPath: '.vibex/specs/L2-skeleton/p.yaml',
			})
		).toEqual([]);
	});

	it('resolveSpecRefToPath uses exact spec.name in catalog before heuristic path', () => {
		const catalog = [
			{ path: '.vibex/specs/L3-module/MOD-custom-path.yaml', name: 'MOD-build-panel' },
		];
		expect(resolveSpecRefToPath('MOD-build-panel', null, catalog)).toBe(
			'.vibex/specs/L3-module/MOD-custom-path.yaml'
		);
	});

	it('resolveSpecRefToPath catalog requires exact string match on spec.name', () => {
		const catalog = [{ path: '.vibex/specs/L4-feature/Z.yaml', name: 'FEAT-only-catalog' }];
		expect(resolveSpecRefToPath('FEAT-only-catalog', null, catalog)).toBe('.vibex/specs/L4-feature/Z.yaml');
		expect(resolveSpecRefToPath('feat-only-catalog', null, catalog)).not.toBe(
			'.vibex/specs/L4-feature/Z.yaml'
		);
	});
});
