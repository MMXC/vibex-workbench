import { json } from '@sveltejs/kit';
import fs from 'fs';
import path from 'path';
import { CANONICAL_SPEC_SUBDIRS } from '$lib/workbench/spec-layout-dirs';

export async function POST(event) {
	const body = await event.request.json().catch(() => ({}));
	const workspaceRoot = path.resolve(body.workspace_root || body.workspaceRoot || '');
	const confirm = body.confirm === true;

	if (!workspaceRoot) {
		return json({ ok: false, error: 'workspace_root required' }, { status: 400 });
	}
	if (!fs.existsSync(workspaceRoot)) {
		return json({ ok: false, error: 'Directory does not exist' }, { status: 400 });
	}
	if (!confirm) {
		return json({
			ok: false,
			error: 'scaffold requires confirm=true',
			would_create: ['.vibex/specs/', 'Makefile', 'generators/gen.py', 'spec-templates/', '.memlace/'],
		});
	}

	const created: string[] = [];
	const errors: string[] = [];

	for (const rel of CANONICAL_SPEC_SUBDIRS) {
		const p = path.join(workspaceRoot, ...rel.split('/'));
		if (!fs.existsSync(p)) {
			try {
				fs.mkdirSync(p, { recursive: true });
				created.push(rel + '/');
			} catch (e: unknown) {
				const msg = e instanceof Error ? e.message : String(e);
				errors.push(`mkdir ${rel}: ${msg}`);
			}
		}
	}

	for (const dir of ['generators', 'spec-templates']) {
		const p = path.join(workspaceRoot, dir);
		if (!fs.existsSync(p)) {
			try {
				fs.mkdirSync(p, { recursive: true });
				created.push(dir + '/');
			} catch (e: unknown) {
				const msg = e instanceof Error ? e.message : String(e);
				errors.push(`mkdir ${dir}: ${msg}`);
			}
		}
	}

	// Create Makefile
	const makefilePath = path.join(workspaceRoot, 'Makefile');
	if (!fs.existsSync(makefilePath)) {
		const makefile = `# VibeX Workbench Makefile
WORKSPACE ?= .

.PHONY: generate lint-specs validate verify-specs

generate:
\t@echo "Run: python3 generators/gen.py --workspace $(WORKSPACE)"
\t@python3 generators/gen.py --workspace $(WORKSPACE)

lint-specs validate:
\t@python3 generators/validate_specs.py --workspace $(WORKSPACE)

verify-specs:
\t@echo "Building verify_specs..."
\t@cd $(WORKSPACE) && go build -o verify_specs ./cmd/verify_specs/
\t@./verify_specs --workspace $(WORKSPACE) --format short
`;
		try {
			fs.writeFileSync(makefilePath, makefile);
			created.push('Makefile');
		} catch (e: unknown) {
			const msg = e instanceof Error ? e.message : String(e);
			errors.push(`write Makefile: ${msg}`);
		}
	}

	// Create generators/gen.py stub
	const genPyPath = path.join(workspaceRoot, 'generators', 'gen.py');
	if (!fs.existsSync(genPyPath)) {
		const genPy = `#!/usr/bin/env python3
"""VibeX Code Generator — stub"""
import argparse, sys

parser = argparse.ArgumentParser(description='VibeX Code Generator')
parser.add_argument('--workspace', default='.', help='Workspace root')
args = parser.parse_args()

print(f"[gen] workspace={args.workspace}")
print("[gen] Generator stub — implement your spec-to-code logic here")
sys.exit(0)
`;
		try {
			fs.writeFileSync(genPyPath, genPy);
			fs.chmodSync(genPyPath, 0o755);
			created.push('generators/gen.py');
		} catch (e: unknown) {
			const msg = e instanceof Error ? e.message : String(e);
			errors.push(`write generators/gen.py: ${msg}`);
		}
	}

	// Create a sample L1 spec
	const sampleL1 = path.join(workspaceRoot, '.vibex', 'specs', 'L1-goal', 'CONCEPT-sample.yaml');
	if (!fs.existsSync(sampleL1)) {
		const sample = `---
spec:
  name: CONCEPT-sample
  level: L1
  status: draft
  parent: ""
meta:
  module: sample
  owner: ""
lifecycle:
  current: concept
  updated: ${new Date().toISOString().split('T')[0]}
display:
  title: Sample Concept
  summary: ""
structure:
  parent: ""
  children: []
  dependencies: []
  impacted_files: []
content: {}
`;
		try {
			fs.writeFileSync(sampleL1, sample);
			created.push('.vibex/specs/L1-goal/CONCEPT-sample.yaml');
		} catch (e: unknown) {
			const msg = e instanceof Error ? e.message : String(e);
			errors.push(`write sample spec: ${msg}`);
		}
	}

	return json({ ok: errors.length === 0, created, errors, workspace: workspaceRoot });
}
