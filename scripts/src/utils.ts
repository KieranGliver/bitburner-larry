import type { NS, ScriptArg, Server } from '@ns';
import type { process } from './data';

/**
 * Returns a list of servers the player can access based on available port opening programs.
 * @remark Ram Cost: 1.15 GB
 * - ns.getPurchasedServers(): 1.05 GB
 * - ns.fileExists(): 0.1 GB
 * @param ns - The Netscript environment.
 * @param allFlag - If true, includes all servers regardless of access level.
 * @returns An array of accessible server hostnames.
 */
export function getServerList(ns: NS, allFlag = false): Array<string> {
	// Arrays of all servers divided by number of ports required.
	const servers0Port = [
		'n00dles',
		'foodnstuff',
		'sigma-cosmetics',
		'joesguns',
		'nectar-net',
		'hong-fang-tea',
		'harakiri-sushi',
	];

	const servers1Port = ['neo-net', 'CSEC', 'zer0', 'max-hardware', 'iron-gym'];

	const servers2Port = [
		'phantasy',
		'silver-helix',
		'omega-net',
		'avmnite-02h',
		'crush-fitness',
		'johnson-ortho',
		'the-hub',
	];

	const servers3Port = [
		'netlink',
		'rothman-uni',
		'summit-uni',
		'rho-construction',
		'I.I.I.I',
		'millenium-fitness',
		'computek',
		'catalyst',
	];

	const servers4Port = [
		'unitalife',
		'univ-energy',
		'zb-def',
		'applied-energetics',
		'run4theh111z',
		'syscore',
		'lexo-corp',
		'aevum-police',
		'global-pharm',
		'nova-med',
		'.',
		'alpha-ent',
		'snap-fitness',
	];

	const servers5Port = [
		'zb-institute',
		'galactic-cyber',
		'deltaone',
		'icarus',
		'defcomm',
		'infocomm',
		'microdyne',
		'stormtech',
		'kuai-gong',
		'b-and-a',
		'nwo',
		'megacorp',
		'vitalife',
		'4sigma',
		'blade',
		'omnia',
		'solaris',
		'zeus-med',
		'taiyang-digital',
		'titan-labs',
		'fulcrumtech',
		'helios',
		'powerhouse-fitness',
		'omnitek',
		'clarkinc',
		'ecorp',
		'fulcrumassets',
		'The-Cave',
		'aerocorp',
	];

	//if (ns.getServer("w0r1d_d43m0n")) {
	//    servers5Port.push("w0r1d_d43m0n");
	//}

	const ret: string[] = [];

	ret.push(...ns.getPurchasedServers().concat(servers0Port));

	// checks if have port openning file on home. Adds servers that can be accessed.
	if (ns.fileExists('BruteSSH.exe', 'home') || allFlag) {
		ret.push(...servers1Port);
	}
	if (ns.fileExists('FTPCrack.exe', 'home') || allFlag) {
		ret.push(...servers2Port);
	}
	if (ns.fileExists('relaySMTP.exe', 'home') || allFlag) {
		ret.push(...servers3Port);
	}
	if (ns.fileExists('HTTPWorm.exe', 'home') || allFlag) {
		ret.push(...servers4Port);
	}
	if (ns.fileExists('SQLInject.exe', 'home') || allFlag) {
		ret.push(...servers5Port);
	}

	ret.push('home');

	// Return list with home computer at end.
	return ret;
}

/**
 * Crack a server by running all available hacking programs on it.
 * @remark Ram Cost: 0.5 GB
 * - ns.getServerNumPortsRequired: 0.1 GB
 * - ns.fileExists: 0.1 GB
 * - ns.sqlinject: 0.05 GB
 * - ns.httpworm: 0.05 GB
 * - ns.relaysmtp: 0.05 GB
 * - ns.ftpcrack: 0.05 GB
 * - ns.brutessh: 0.05 GB
 * - ns.nuke: 0.05 GB
 * @param ns - Netscript object
 * @param server - The server to crack, either as a string or a Server object.
 */
export function crackServer(ns: NS, server: string | Server) {
	const portNum =
		typeof server === 'string'
			? ns.getServerNumPortsRequired(server)
			: ns.getServerNumPortsRequired(server.hostname);
	const hostname = typeof server === 'string' ? server : server.hostname;

	if (portNum > 4 && ns.fileExists('SQLInject.exe', 'home')) {
		ns.sqlinject(hostname);
	}
	if (portNum > 3 && ns.fileExists('HTTPWorm.exe', 'home')) {
		ns.httpworm(hostname);
	}
	if (portNum > 2 && ns.fileExists('relaySMTP.exe', 'home')) {
		ns.relaysmtp(hostname);
	}
	if (portNum > 1 && ns.fileExists('FTPCrack.exe', 'home')) {
		ns.ftpcrack(hostname);
	}
	if (portNum > 0 && ns.fileExists('BruteSSH.exe', 'home')) {
		ns.brutessh(hostname);
	}

	ns.nuke(hostname);
}

/**
 * Attempts to run a script with multiple threads on multiple servers
 * @remark Ram Cost: 2.1 GB
 * - ns.exec(): 1.3 GB
 * - ns.scp(): 0.6 GB
 * - ns.getScriptRam(): 0.1 GB
 * - ns.getServerMaxRam(): 0.05 GB
 * - ns.getServerUsedRam(): 0.05 GB
 * @param ns - Netscript object
 * @param serverList - list of servers to try to run the script on
 * @param script - script to run
 * @param threads - number of threads to run the script with
 * @param args - arguments to pass to the script
 * @returns pid[] of the script that was run
 */
export function runScript(
	ns: NS,
	serverList: string[],
	script: string,
	threads: number,
	...args: ScriptArg[]
): process[] {
	// Disable all logs
	ns.disableLog('disableLog');
	ns.disableLog('enableLog');
	ns.disableLog('getServerMaxRam');
	ns.disableLog('getServerUsedRam');
	ns.disableLog('exec');
	ns.disableLog('scp');

	const ret: process[] = [];
	let n = threads;

	for (const hostname of serverList) {
		if (n === 0) {
			break;
		}

		const availableRam =
			ns.getServerMaxRam(hostname) - ns.getServerUsedRam(hostname);
		const scriptRam = ns.getScriptRam(script);
		const serverThreads = Math.max(
			Math.min(Math.floor(availableRam / scriptRam), n),
			0,
		);

		if (serverThreads) {
			ns.scp('utils.js', hostname);
			ns.scp('data.js', hostname);
			ns.scp(script, hostname);

			const pid = ns.exec(script, hostname, serverThreads, ...args);

			if (pid) {
				ret.push({
					pid: pid,
					arguments: [...args],
					script: script,
					server: hostname,
					threads: serverThreads,
				});
			}
		}

		n -= serverThreads;
	}

	// Enable all logs
	ns.enableLog('getServerMaxRam');
	ns.enableLog('getServerUsedRam');
	ns.enableLog('exec');
	ns.enableLog('scp');
	ns.enableLog('disableLog');
	ns.enableLog('enableLog');

	if (n > 0) {
		ns.print(
			`Warn: Not enough servers available to run ${script} with ${threads} threads.`,
		);
		//ns.ui.openTail();
	}

	ns.print(
		`Finished running ${script} with ${threads - n} of ${threads} threads with args: ${[...args]}`,
	);

	return ret;
}
/**
 * Generates a formatted UI string with a customizable outline and content.
 * The UI can display either a single string or an array of strings, centered within a bordered box.
 *
 * @remark 0 GB
 * @param data - The content to display inside the UI. Can be a single string or an array of strings.
 * @param width - The total width of the UI box, including the outline.
 * @param thickness - The thickness of the outline border. Defaults to 1.
 * @param outlinePattern - The pattern used for the outline border. Defaults to "#".
 * @returns A string representing the formatted UI box.
 */
export function createUI(
	data: string | Array<string>,
	width: number,
	thickness = 1,
	outlinePattern = '#',
): string {
	let ui = `${outlinePattern
		.repeat(Math.max(Math.ceil(width / outlinePattern.length), 0))
		.substring(0, width)}\n`;

	if (typeof data === 'string') {
		ui = `${
			ui +
			outlinePattern
				.repeat(Math.max(Math.ceil(thickness / outlinePattern.length), 0))
				.substring(0, thickness) +
			' '.repeat(
				Math.max(Math.floor((width - 2 * thickness - data.length) / 2), 0),
			) +
			data +
			' '.repeat(
				Math.max(Math.ceil((width - 2 * thickness - data.length) / 2), 0),
			) +
			outlinePattern
				.repeat(Math.max(Math.ceil(thickness / outlinePattern.length), 0))
				.substring(0, thickness)
		}\n`;
	} else {
		for (const d of data) {
			ui = `${
				ui +
				outlinePattern
					.repeat(Math.max(Math.ceil(thickness / outlinePattern.length), 0))
					.substring(0, thickness) +
				' '.repeat(
					Math.max(Math.floor((width - 2 * thickness - d.length) / 2), 0),
				) +
				d +
				' '.repeat(
					Math.max(Math.ceil((width - 2 * thickness - d.length) / 2), 0),
				) +
				outlinePattern
					.repeat(Math.max(Math.ceil(thickness / outlinePattern.length), 0))
					.substring(0, thickness)
			}\n`;
		}
	}

	return `${
		ui +
		outlinePattern
			.repeat(Math.ceil(width / outlinePattern.length))
			.substring(0, width)
	}\n`;
}

/**
 * displays a message in the UI and sleeps for a specified amount of time
 *
 * @remark 0 GB
 * @param ns - Netscript object
 * @param text - The text to display in the UI. If empty, the UI will be cleared.
 * @param sleepLength - The amount of time to sleep in milliseconds.
 */
export async function displayUI(ns: NS, text: string, sleepLength: number, size: {x: number, y: number} = {x: 0, y: 0}) {
	ns.disableLog('sleep');
	ns.clearLog();
	if (text) {
		ns.print(text);
	}
	await ns.sleep(sleepLength);
	// resize the UI to fit the text
	if (size.x > 0 && size.y > 0) {
		ns.ui.resizeTail(size.x*9.635, size.y*25.548);
	}
	ns.enableLog('sleep');
}
