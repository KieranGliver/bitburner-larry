import type { ScriptArg } from '@ns';

export const growFile = 'grow.js';
export const weakFile = 'weak.js';
export const hackFile = 'hack.js';

export interface process {
	pid: number;
	arguments: ScriptArg[];
	script: string;
	server: string;
	threads: number;
}
