import { NS } from "@ns";

interface ExecRequest {
  id: string;
  action: "exec";
  server: string;
  script: string;
  threads: number;
  args: (string | number | boolean)[];
}

interface ColResponse {
  id: string;
  success: boolean;
  pid: number;
  error: string;
}

export async function main(ns: NS): Promise<void> {
  ns.disableLog("ALL");
  ns.print("Col daemon started, watching /inbox/");

  while (true) {
    const inboxFiles = ns.ls("home", "/inbox/");

    for (const file of inboxFiles) {
      const content = ns.read(file);
      if (!content) continue;

      let req: ExecRequest;
      try {
        req = JSON.parse(content);
      } catch {
        ns.print(`ERROR: failed to parse ${file}`);
        ns.rm(file);
        continue;
      }

      const response: ColResponse = { id: req.id, success: false, pid: 0, error: "" };

      if (req.action === "exec") {
        const pid = ns.exec(req.script, req.server, req.threads ?? 1, ...(req.args ?? []));
        if (pid > 0) {
          response.success = true;
          response.pid = pid;
        } else {
          response.error = `exec failed: not enough RAM or script not found`;
        }
      } else {
        response.error = `unknown action: ${(req as ExecRequest).action}`;
      }

      ns.write(`/outbox/${req.id}.txt`, JSON.stringify(response), "w");
      ns.rm(file);
      ns.print(`${req.id}: ${response.success ? `pid ${response.pid}` : response.error}`);
    }

    await ns.sleep(500);
  }
}
