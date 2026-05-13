import { describe, expect, it } from 'vitest';
import {
	appendChildrenPaths,
	collectL3CandidatesFromL2,
	collectL5CandidatesFromL4,
	inferSpawnLayer,
	replaceBracePlaceholders,
	renderL3ModuleFromTemplate,
} from './spawn-child-specs';

const sampleL2 = `spec:
  level: 2_skeleton
  name: demo-l2
structure:
  children:
  - .vibex/specs/L3-module/MOD-agent.yaml
content:
  l2_l3_lineage:
    which_modules_become_l3:
    - module: agent
      l3_note: 已在 children 中挂载
    - module: backend
      l3_note: API 层
    - module: ide-chrome
      l3_note: UI 壳
`;

const sampleL4 = `spec:
  level: 4_feature
  name: FEAT-make-integration
structure:
  children: []
content:
  behaviors:
  - id: B1
    trigger: 点击 lint
  - id: B2
    trigger: 点击 generate
`;

describe('spawn-child-specs', () => {
	it('inferSpawnLayer L2/L4', () => {
		expect(inferSpawnLayer(sampleL2)).toBe('L2');
		expect(inferSpawnLayer(sampleL4)).toBe('L4');
		expect(inferSpawnLayer('spec:\n  level: 3_module\n')).toBe(null);
	});

	it('collectL3CandidatesFromL2 filters linked paths', () => {
		const cs = collectL3CandidatesFromL2(sampleL2, 'demo-l2');
		const agent = cs.find(c => c.specName === 'MOD-agent');
		expect(agent?.alreadyLinked).toBe(true);
		const backend = cs.find(c => c.specName === 'MOD-backend');
		expect(backend?.alreadyLinked).toBe(false);
		expect(backend?.relativePath).toBe('.vibex/specs/L3-module/MOD-backend.yaml');
	});

	it('collectL5CandidatesFromL4 builds slice names', () => {
		const cs = collectL5CandidatesFromL4(sampleL4, 'FEAT-make-integration');
		expect(cs.map(c => c.specName)).toEqual([
			'SLICE-make-integration-B1',
			'SLICE-make-integration-B2',
		]);
		expect(cs[0]?.behaviorId).toBe('B1');
	});

	it('appendChildrenPaths merges unique', () => {
		const parent = `spec:
  name: p
structure:
  children:
  - .vibex/specs/L3-module/MOD-a.yaml
`;
		const next = appendChildrenPaths(parent, [
			'.vibex/specs/L3-module/MOD-b.yaml',
			'.vibex/specs/L3-module/MOD-a.yaml',
		]);
		expect(next).toContain('MOD-a.yaml');
		expect(next).toContain('MOD-b.yaml');
	});

	it('replaceBracePlaceholders prefers longer keys first', () => {
		const out = replaceBracePlaceholders('a {foo-bar} {foo}', { 'foo-bar': 'X', foo: 'Y' });
		expect(out).toBe('a X Y');
	});

	it('renderL3ModuleFromTemplate fills core tokens', () => {
		const tpl = `spec:
  name: "{module-name}"
  parent: "{l2-spec-name}"
`;
		const rendered = renderL3ModuleFromTemplate(tpl, {
			moduleSpecName: 'MOD-backend',
			l2SpecName: 'demo-l2',
			owner: 'hermes',
			dateYmd: '2026-05-08',
			titleZh: '后端',
			summaryLine: 'API',
			descriptionParagraph: 'desc',
		});
		expect(rendered).toContain('MOD-backend');
		expect(rendered).toContain('demo-l2');
	});
});
