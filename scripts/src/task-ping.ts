import type { NS } from "@ns";

export async function main(ns: NS): Promise<void> {
	const id = ns.args[0] as string;
	await fetch("http://localhost:12525/done", {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ id, success: true, message: "pong" }),
	});
}
