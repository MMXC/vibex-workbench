import { json } from '@sveltejs/kit';
import fs from 'fs';
import path from 'path';

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
			would_create: ['specs/', 'Makefile', 'generators/gen.py', 'spec-templates/', '.memlace/'],
		});
	}

	const created: string[] = [];
	const errors: string[] = [];

	// Create directories
	for (const dir of ['specs', 'generators', 'spec-templates', 'specs/L1-concept', 'specs/L2-requirement', 'specs/L3-module', 'specs/L4-feature', 'specs/L5-slice']) {
		const p = path.join(workspaceRoot, dir);
		if (!fs.existsSync(p)) {
			try {
				fs.mkdirSync(p, { recursive: true });
				created.push(dir + '/');
			} catch (e: any) {
				errors.push(`mkdir ${dir}: ${e.message}`);
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
		} catch (e: any) {
			errors.push(`write Makefile: ${e.message}`);
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
		} catch (e: any) {
			errors.push(`write generators/gen.py: ${e.message}`);
		}
	}

	// Create a sample L1 spec
	const sampleL1 = path.join(workspaceRoot, 'specs', 'L1-concept', 'CONCEPT-sample.yaml');
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
			created.push('specs/L1-concept/CONCEPT-sample.yaml');
		} catch (e: any) {
			errors.push(`write sample spec: ${e.message}`);
		}
	}

	return json({ ok: errors.length === 0, created, errors, workspace: workspaceRoot });
}
