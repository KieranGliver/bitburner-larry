import type { NS } from "@ns";

export async function main(ns: NS): Promise<void> {
	const id = ns.args[0] as string;
	const budgetFraction = ns.args[1] as number;

	const money = ns.getServerMoneyAvailable("home");
	const budget = money * budgetFraction;
	const limit = ns.getPurchasedServerLimit();
	const maxRam = ns.getPurchasedServerMaxRam();
	const purchased = ns.getPurchasedServers();
	const bought: string[] = [];
	const upgraded: string[] = [];

	function findMaxRam(b: number): number {
		let ram = 2;
		while (ram * 2 <= maxRam && ns.getPurchasedServerCost(ram * 2) <= b) ram *= 2;
		return ns.getPurchasedServerCost(ram) <= b ? ram : 0;
	}

	if (purchased.length < limit) {
		const ram = findMaxRam(budget);
		if (ram >= 2) {
			let idx = 0;
			while (purchased.includes(`pserv-${idx}`)) idx++;
			const hostname = ns.purchaseServer(`pserv-${idx}`, ram);
			if (hostname) bought.push(hostname);
		}
	} else {
		// upgrade smallest
		let smallest = purchased[0];
		let smallestRam = ns.getServer(smallest).maxRam;
		for (const h of purchased) {
			const r = ns.getServer(h).maxRam;
			if (r < smallestRam) {
				smallest = h;
				smallestRam = r;
			}
		}
		const nextRam = smallestRam * 2;
		if (nextRam <= maxRam && ns.getPurchasedServerCost(nextRam) <= budget) {
			if (ns.upgradePurchasedServer(smallest, nextRam)) upgraded.push(smallest);
		}
	}

	await fetch("http://localhost:12525/done", {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ id, success: true, error: "", bought, upgraded }),
	});
}
