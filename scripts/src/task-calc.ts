import type { NS } from "@ns";

export async function main(ns: NS): Promise<void> {
	const id = ns.args[0] as string;
	const target = ns.args[1] as string;
	const hackPercent = ns.args[2] as number;

	const maxMoney = ns.getServerMaxMoney(target);
	// Use hackAnalyze (fraction per thread) to compute threads for max-money state.
	// hackAnalyzeThreads returns -1 when hackAmount > currentMoney, so we avoid it.
	const hackFracPerThread = ns.hackAnalyze(target);
	const hackThreads = hackFracPerThread > 0 ? Math.ceil(hackPercent / hackFracPerThread) : 1;
	// ceil means we steal slightly more than hackPercent — use the actual fraction
	// stolen by hackThreads so grow compensates for what hack really takes
	const actualHackFrac = Math.min(hackThreads * hackFracPerThread, 0.99);
	const growMult = 1 / (1 - actualHackFrac);
	const growThreads = Math.ceil(ns.growthAnalyze(target, growMult));
	const weakenPer = ns.weakenAnalyze(1);
	const weakenHackThreads = Math.ceil(ns.hackAnalyzeSecurity(hackThreads, target) / weakenPer);
	const weakenGrowThreads = Math.ceil(ns.growthAnalyzeSecurity(growThreads, target) / weakenPer);
	const hackTime = ns.getHackTime(target);
	const growTime = ns.getGrowTime(target);
	const weakenTime = ns.getWeakenTime(target);

	const currentSecurity = ns.getServerSecurityLevel(target);
	const minSecurity = ns.getServerMinSecurityLevel(target);
	const prepWeakenThreads = Math.ceil(Math.max(0, currentSecurity - minSecurity) / weakenPer);

	const currentMoney = ns.getServerMoneyAvailable(target);
	const prepGrowMult = maxMoney / Math.max(currentMoney, 1);
	const prepGrowThreads = currentMoney >= maxMoney ? 0 : Math.ceil(ns.growthAnalyze(target, prepGrowMult));
	const prepGrowWeakenThreads = prepGrowThreads === 0 ? 0 : Math.ceil(ns.growthAnalyzeSecurity(prepGrowThreads, target) / weakenPer);

	await fetch("http://localhost:12525/done", {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({
			id,
			success: true,
			error: "",
			target,
			hackPercent,
			prepWeakenThreads,
			prepGrowThreads,
			prepGrowWeakenThreads,
			hackThreads,
			growThreads,
			weakenHackThreads,
			weakenGrowThreads,
			hackTime,
			growTime,
			weakenTime,
		}),
	});
}
