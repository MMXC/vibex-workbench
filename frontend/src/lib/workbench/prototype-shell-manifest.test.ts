import { describe, expect, it } from 'vitest';
import {
	buildFeatureShellRoute,
	deriveEntryHtmlForFeature,
	getPrototypeFileRel,
	slugFromSpecName,
} from './prototype-shell-manifest';

describe('prototype-shell-manifest', () => {
	it('slugFromSpecName normalizes FEAT names', () => {
		expect(slugFromSpecName('FEAT-spec-prototype-shell-deck')).toBe('feat-spec-prototype-shell-deck');
	});

	it('getPrototypeFileRel reads prototype.file', () => {
		const y = `spec:
  name: X
prototype:
  file: .vibex/prototypes/foo.html
`;
		expect(getPrototypeFileRel(y)).toBe('.vibex/prototypes/foo.html');
	});

	it('deriveEntryHtmlForFeature prefers yaml then default', () => {
		const y = `prototype:\n  file: .vibex/prototypes/a.html\n`;
		expect(
			deriveEntryHtmlForFeature({
				specPath: 'specs/L4-feature/FEAT-x.yaml',
				yamlContent: y,
			})
		).toBe('.vibex/prototypes/a.html');
		expect(
			deriveEntryHtmlForFeature({
				specPath: 'specs/L4-feature/FEAT-x.yaml',
				yamlContent: 'spec:\n  name: X\n',
			})
		).toBe('.vibex/prototypes/FEAT-x.html');
	});

	it('buildFeatureShellRoute wires fields', () => {
		const r = buildFeatureShellRoute({
			specName: 'FEAT-foo',
			specPath: 'specs/L4-feature/FEAT-foo.yaml',
			displayTitle: 'Foo',
			yamlContent: '',
		});
		expect(r.specRef).toBe('FEAT-foo');
		expect(r.id).toBe('feat-foo');
		expect(r.path).toBe('/proto/feat-foo');
		expect(r.kind).toBe('feature');
	});
});
