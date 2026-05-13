/** Canonical spec / agent buckets under workspace root (POSIX). Sync with pkg/speclayout/layout.go */
export const CANONICAL_SPEC_SUBDIRS = [
	'.vibex/specs/L1-goal',
	'.vibex/specs/L2-skeleton',
	'.vibex/specs/L3-module',
	'.vibex/specs/L4-feature',
	'.vibex/specs/L5-slice',
	'.vibex/specs/_governance',
	'.vibex/agents/flows',
	'.vibex/prototypes',
] as const;

/** 规格树根目录（相对工作区根，POSIX）。 */
export const VIBEX_SPECS_ROOT = '.vibex/specs';
