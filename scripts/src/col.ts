import type { NS } from "@ns";

interface ExecRequest {
	id: string;
	action: "exec";
	server: string;
	script: string;
	threads: number;
	args: (string | number | boolean)[];
}

interface CrackRequest {
	id: string;
	action: "crack";
	targets: string[]; // empty = crack all servers
}

interface DeployRequest {
	id: string;
	action: "deploy";
	server: string;
	script: string;
	threads: number;
	args: (string | number | boolean)[];
}

interface KillAllRequest {
	id: string;
	action: "killall";
	servers: string[]; // empty = all servers; col.js on home is always preserved
}


type ColRequest = ExecRequest | CrackRequest | DeployRequest | KillAllRequest;

interface ExecResponse {
  id: string;
	success: boolean;
	pid: number;
	error: string;
}

interface CrackResponse {
	id: string;
	success: boolean;
	cracked: string[];
	failed: string[];
	error: string;
}

interface KillAllResponse {
	id: string;
	success: boolean;
	killed: string[]; // servers that had scripts stopped
	error: string;
}

export function getServerList(ns) {
	const serversPurchased = ns.getPurchasedServers();

	const servers0Port = [
		"n00dles",
		"foodnstuff",
		"sigma-cosmetics",
		"joesguns",
		"nectar-net",
		"hong-fang-tea",
		"harakiri-sushi",
	];

	const servers1Port = ["neo-net", "CSEC", "zer0", "max-hardware", "iron-gym"];

	const servers2Port = [
		"phantasy",
		"silver-helix",
		"omega-net",
		"avmnite-02h",
		"crush-fitness",
		"the-hub",
		"johnson-ortho",
	];

	const servers3Port = [
		"comptek",
		"I.I.I.I",
		"rothman-uni",
		"netlink",
		"catalyst",
		"summit-uni",
		"rho-construction",
		"millenium-fitness",
	];

	const servers4Port = [
		"aevum-police",
		"alpha-ent",
		".",
		"run4theh111z",
		"syscore",
		"lexo-corp",
		"snap-fitness",
		"global-pharm",
		"nova-med",
		"unitalife",
		"applied-energetics",
		"zb-def",
		"univ-energy",
	];

	const servers5Port = [
		"zb-institute",
		"helios",
		"solaris",
		"vitalife",
		"zeus-med",
		"microdyne",
		"titan-labs",
		"omnia",
		"deltaone",
		"defcomm",
		"galactic-cyber",
		"icarus",
		"aerocorp",
		"infocomm",
		"The-Cave",
		"blade",
		"taiyang-digital",
		"stormtech",
		"powerhouse-fitness",
		"clarkinc",
		"omnitek",
		"4sigma",
		"b-and-a",
		"fulcrumassets",
		"fulcrumtech",
		"kuai-gong",
		"nwo",
		"megacorp",
		"ecorp",
	];

	var serversArr: string[] = [];
	if (serversPurchased.length > 0) serversArr = serversArr.concat(serversPurchased);
	serversArr = serversArr.concat(servers5Port.reverse());
	serversArr = serversArr.concat(servers4Port.reverse());
	serversArr = serversArr.concat(servers3Port.reverse());
	serversArr = serversArr.concat(servers2Port.reverse());
	serversArr = serversArr.concat(servers1Port.reverse());
	serversArr = serversArr.concat(servers0Port.reverse());
	serversArr = serversArr.concat("home");
	return serversArr;
}

function crackServer(ns: NS, host: string): boolean {
	if (ns.hasRootAccess(host)) return true;
	const ports = ns.getServerNumPortsRequired(host);
	if (ports > 0) ns.brutessh(host);
	if (ports > 1) ns.ftpcrack(host);
	if (ports > 2) ns.relaysmtp(host);
	if (ports > 3) ns.httpworm(host);
	if (ports > 4) ns.sqlinject(host);
	ns.nuke(host);
	return ns.hasRootAccess(host);
}

export async function main(ns: NS): Promise<void> {
	ns.disableLog("ALL");
	ns.print("Col daemon started, watching /inbox/");

	while (true) {
		const inboxFiles = ns.ls("home", "/inbox/");

		for (const file of inboxFiles) {
			const content = ns.read(file);
			if (!content) continue;

			let req: ColRequest;
			try {
				req = JSON.parse(content);
			} catch {
				ns.print(`ERROR: failed to parse ${file}`);
				ns.write(file, "", "w");
				continue;
			}

			if (req.action === "exec") {
				const response: ExecResponse = {
					id: req.id,
					success: false,
					pid: 0,
					error: "",
				};
				if (!req.script.endsWith(".js") && !req.script.endsWith(".script")) {
					response.error = `invalid script: ${req.script} (must be .js or .script)`;
				} else {
					try {
						const pid = ns.exec(req.script, req.server, req.threads ?? 1, ...(req.args ?? []));
						if (pid > 0) {
							response.success = true;
							response.pid = pid;
						} else {
							response.error = `exec returned 0: not enough RAM or script not found on ${req.server}`;
						}
					} catch (err) {
						response.error = String(err);
					}
				}
				ns.write(`/outbox/${req.id}.txt`, JSON.stringify(response), "w");
				ns.print(`${req.id}: ${response.success ? `pid ${response.pid}` : response.error}`);
			} else if (req.action === "deploy") {
				const response: ExecResponse = {
					id: req.id,
					success: false,
					pid: 0,
					error: "",
				};
				if (!req.script.endsWith(".js") && !req.script.endsWith(".script")) {
					response.error = `invalid script: ${req.script} (must be .js or .script)`;
				} else {
					try {
						const copied = ns.scp(req.script, req.server, "home");
						if (!copied) {
							response.error = `scp failed: could not copy ${req.script} to ${req.server}`;
						} else {
							const pid = ns.exec(req.script, req.server, req.threads ?? 1, ...(req.args ?? []));
							if (pid > 0) {
								response.success = true;
								response.pid = pid;
							} else {
								response.error = `exec returned 0: not enough RAM or script not found on ${req.server}`;
							}
						}
					} catch (err) {
						response.error = String(err);
					}
				}
				ns.write(`/outbox/${req.id}.txt`, JSON.stringify(response), "w");
				ns.print(`${req.id}: ${response.success ? `pid ${response.pid}` : response.error}`);
			} else if (req.action === "killall") {
				const response: KillAllResponse = {
					id: req.id,
					success: true,
					killed: [],
					error: "",
				};
				const targets = req.servers.length > 0 ? req.servers : ["home", ...getServerList(ns)];
				for (const host of targets) {
					try {
						if (host === "home") {
							// preserve col.js — kill everything else individually
							const procs = ns.ps("home");
							let any = false;
							for (const proc of procs) {
								if (proc.filename === "col.js") continue;
								ns.kill(proc.pid);
								any = true;
							}
							if (any) response.killed.push("home");
						} else {
							if (ns.killall(host)) response.killed.push(host);
						}
					} catch {
						// skip invalid/inaccessible hosts silently
					}
				}
				ns.write(`/outbox/${req.id}.txt`, JSON.stringify(response), "w");
				ns.print(
					`${req.id}: killall — stopped scripts on: ${response.killed.join(", ") || "none"}`,
				);
			} else if (req.action === "crack") {
				const targets = req.targets.length > 0 ? req.targets : getServerList(ns);
				const cracked: string[] = [];
				const failed: string[] = [];
				for (const host of targets) {
					try {
						if (crackServer(ns, host)) cracked.push(host);
						else failed.push(host);
					} catch {
						failed.push(host);
					}
				}
				const response: CrackResponse = {
					id: req.id,
					success: failed.length === 0,
					cracked,
					failed,
					error: "",
				};
				ns.write(`/outbox/${req.id}.txt`, JSON.stringify(response), "w");
				ns.print(`${req.id}: cracked ${cracked.length}, failed ${failed.length}`);
			}

			ns.write(file, "", "w"); // clear processed request; Larry deletes the file via RPC
		}

		await ns.sleep(500);
	}
}
