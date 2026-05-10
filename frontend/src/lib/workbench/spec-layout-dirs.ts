/** Canonical spec buckets under workspace root (POSIX). Sync with pkg/speclayout/layout.go */
export const CANONICAL_SPEC_SUBDIRS = [
	'specs/L1-goal',
	'specs/L2-skeleton',
	'specs/L3-module',
	'specs/L4-feature',
	'specs/L5-slice',
	'specs/_governance',
	'.agents/flows',
] as const;
