// agent-notify:pi-extension
import { spawn } from "node:child_process";

const BINARY = __AGENT_NOTIFY_BINARY__;

function dispatch(type, ctx, body = "") {
  const sessionManager = ctx?.sessionManager;
  const payload = {
    type,
    session_id: sessionManager?.getSessionId?.() || "",
    cwd: ctx?.cwd || process.cwd(),
    body,
  };

  try {
    const child = spawn(BINARY, ["handle-pi-hook"], {
      stdio: ["pipe", "ignore", "ignore"],
      detached: true,
    });
    child.on("error", () => {});
    child.stdin.on("error", () => {});
    child.stdin.end(JSON.stringify(payload));
    child.unref();
  } catch {
    // Notifications must never interrupt Pi.
  }
}

export default function (pi) {
  let pendingOutcome;
  pi.on("session_start", (_event, ctx) => {
    pendingOutcome = undefined;
    dispatch("session_start", ctx);
  });

  pi.on("ui_prompt_start", (event, ctx) => {
    const type = event.kind === "confirm" ? "permission_required" : "input_required";
    const body = event.title || (type === "permission_required" ? "操作需要您的确认" : "等待您的输入");
    dispatch(type, ctx, body);
  });

  pi.on("agent_before_settle", (event) => {
    pendingOutcome = event.outcome;
  });

  pi.on("agent_settled", (_event, ctx) => {
    const outcome = pendingOutcome;
    pendingOutcome = undefined;
    if (outcome === "completed") {
      dispatch("run_completed", ctx);
    } else if (outcome === "error") {
      dispatch("run_failed", ctx, "运行出错");
    }
  });
}
