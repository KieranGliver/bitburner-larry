import type { NS, Server, ProcessInfo } from "@ns";

// Standalone server list — no import from col.js so this script runs on any server.
function getAllHosts(ns: NS): string[] {
	const purchased = ns.getPurchasedServers();
	return [
		...purchased,
		"home",
		"n00dles", "foodnstuff", "sigma-cosmetics", "joesguns", "nectar-net",
		"hong-fang-tea", "harakiri-sushi", "neo-net", "CSEC", "zer0", "max-hardware",
		"iron-gym", "phantasy", "silver-helix", "omega-net", "avmnite-02h",
		"crush-fitness", "the-hub", "johnson-ortho", "comptek", "I.I.I.I",
		"rothman-uni", "netlink", "catalyst", "summit-uni", "rho-construction",
		"millenium-fitness", "aevum-police", "alpha-ent", "run4theh111z", "syscore",
		"lexo-corp", "snap-fitness", "global-pharm", "nova-med", "unitalife",
		"applied-energetics", "zb-def", "univ-energy", "zb-institute", "helios",
		"solaris", "vitalife", "zeus-med", "microdyne", "titan-labs", "omnia",
		"deltaone", "defcomm", "galactic-cyber", "icarus", "aerocorp", "infocomm",
		"The-Cave", "blade", "taiyang-digital", "stormtech", "powerhouse-fitness",
		"clarkinc", "omnitek", "4sigma", "b-and-a", "fulcrumassets", "fulcrumtech",
		"kuai-gong", "nwo", "megacorp", "ecorp",
	];
}

export async function main(ns: NS): Promise<void> {
	const id = ns.args[0] as string;
	const { bitNodeN: _bitNodeN, ...player } = ns.getPlayer() as any;
	const servers: (Server & { processes: ProcessInfo[] })[] = [];

	const hosts = new Set<string>(getAllHosts(ns));
	for (const host of hosts) {
		try {
			servers.push({ ...ns.getServer(host), processes: ns.ps(host) });
		} catch {
			/* skip inaccessible host */
		}
	}

	await fetch("http://localhost:12525/done", {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ id, success: true, error: "", player, servers }),
	});
}
