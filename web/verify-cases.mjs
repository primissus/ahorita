import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "..");
const publicDir = path.join(__dirname, "public");

// 24+ expressions from CASES.md covering dates, arithmetic, relative, spans,
// recurrence, cron, counting, timezones, business days, holidays, and formatting.
const CASES_EXPRESSIONS = [
  "today",
  "now - 1d",
  "2026-06-02 + 2d",
  "2026-06-02 - 1h",
  "yesterday",
  "2d ago",
  "last week",
  "closest monday",
  "today - 1w, monday",
  "end of day, today",
  "next month",
  "in Q3",
  "week 2",
  "90m",
  "during 2 days",
  "every monday",
  "every day 2nd",
  "every monday, to cron",
  "how many days between 2026-11-16 and 2026-11-20",
  "2026-11-14 14:30 in Tokyo",
  "now to UTC",
  "2026-11-20 - 2026-11-16, to business days",
  "2026-11-14, next business day",
  "2026-11-13 + 3 business days",
  "2026-12-24 - 2026-11-14",
  "holidays MX 2026",
  "(now - 1 business day) to los_angeles",
  "now to Los Angeles",
  "now to LA",
  "now to california",
  "now to los_angels",
  "2 * (1h + 30m)",
  "today + 3 days",
  "days to 2027-01-01",
  "days to 2027",
  "how many days to 2027",
  "days from 2026-11-16 to 2026-11-20",
  "EoD",
  "SoD",
  "EoM",
  "SoM",
  "EoY",
  "end day",
  "start day",
  "star of day",
  "end minute",
  "yesterday 2pm - now, in hours",
  "yesterday 2pm - now, in minutes",
  "yesterday 2pm - now, in seconds",
  "yesterday 2pm - now, in millis",
  "yesterday 2pm - now, in milliseconds",
  "yesterday 2pm - now, in days",
  "yesterday 2pm - now, in weeks",
  "180s in hours",
  "1500ms in seconds",
  "1y in halfs",
  "1y in quarters",
  "oncall week = from last friday to this friday",
  "oncall week = if today is friday, from today to next friday, else from last friday to this friday",
];

await import(path.join(publicDir, "wasm_exec.js"));
const go = new globalThis.Go();
const wasmBytes = fs.readFileSync(path.join(publicDir, "ahorita.wasm"));
const { instance } = await WebAssembly.instantiate(wasmBytes, go.importObject);
go.run(instance).catch((err) => {
  console.error("WASM runtime failed:", err);
  process.exit(1);
});
assert.equal(typeof globalThis.ahoritaRun, "function");

const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "ahorita-parity-"));
const cliBin = path.join(tmpDir, process.platform === "win32" ? "ahorita.exe" : "ahorita");

try {
  execFileSync("go", ["build", "-o", cliBin, "./cmd/ahorita"], {
    cwd: repoRoot,
    stdio: "inherit",
  });

  // Pass "now" first so WASM uses the exact same wall-clock second that CLI evaluated.
  const stdinPayload = ["now", ...CASES_EXPRESSIONS].join("\n") + "\n";
  const cliConfigPath = path.join(tmpDir, "config.toml");
  fs.writeFileSync(cliConfigPath, "");
  const cliStdout = execFileSync(
    cliBin,
    [
      "--config-file",
      cliConfigPath,
      "--json",
      "--tz",
      "America/Mexico_City",
      "--country",
      "MX",
      "--day-week-start",
      "monday",
    ],
    {
      cwd: repoRoot,
      input: stdinPayload,
      encoding: "utf8",
    }
  );

  const cliResults = cliStdout
    .trim()
    .split("\n")
    .map((line) => JSON.parse(line));
  assert.equal(cliResults.length, CASES_EXPRESSIONS.length + 1);

  const cliNowISO = cliResults[0].iso;
  assert.ok(cliNowISO, "expected non-empty ISO for reference now");

  const baseWasmOpts = {
    now: cliNowISO,
    tz: "America/Mexico_City",
    country: "MX",
    week_start: "monday",
  };

  for (let i = 0; i < CASES_EXPRESSIONS.length; i++) {
    const expr = CASES_EXPRESSIONS[i];
    const cliRes = cliResults[i + 1];
    const wasmRes = JSON.parse(
      globalThis.ahoritaRun(expr, JSON.stringify(baseWasmOpts))
    );
    assert.equal(wasmRes.text, cliRes.text, `text mismatch for "${expr}"`);
    assert.equal(wasmRes.kind, cliRes.kind, `kind mismatch for "${expr}"`);
    assert.equal(wasmRes.iso, cliRes.iso, `iso mismatch for "${expr}"`);
  }

  // Also verify -t relative parity (`now` synced)
  const relCliOut = execFileSync(
    cliBin,
    [
      "--json",
      "--tz",
      "America/Mexico_City",
      "--type",
      "relative",
    ],
    {
      cwd: repoRoot,
      input: "now\ntoday + 2d\nhace 2 días\n",
      encoding: "utf8",
    }
  );
  const [relNow, relFutureCli, relEsCli] = relCliOut
    .trim()
    .split("\n")
    .map((line) => JSON.parse(line));

  const relFutureWasm = JSON.parse(
    globalThis.ahoritaRun(
      "today + 2d",
      JSON.stringify({ now: relNow.iso, tz: "America/Mexico_City", type: "relative" })
    )
  );
  const relEsWasm = JSON.parse(
    globalThis.ahoritaRun(
      "hace 2 días",
      JSON.stringify({ now: relNow.iso, tz: "America/Mexico_City", type: "relative" })
    )
  );
  assert.equal(relFutureWasm.text, relFutureCli.text);
  assert.equal(relEsWasm.text, relEsCli.text);

  // Also verify -f pattern parity (`CASES.md` row 41)
  const fmtCliOut = execFileSync(
    cliBin,
    ["--json", "--tz", "America/Mexico_City", "--format", "dd-MM-yyyy"],
    {
      cwd: repoRoot,
      input: "2026-06-02\n",
      encoding: "utf8",
    }
  );
  const fmtCli = JSON.parse(fmtCliOut.trim());
  const fmtWasm = JSON.parse(
    globalThis.ahoritaRun(
      "2026-06-02",
      JSON.stringify({ now: cliNowISO, tz: "America/Mexico_City", format: "dd-MM-yyyy" })
    )
  );
  assert.equal(fmtWasm.text, fmtCli.text);
  assert.equal(fmtWasm.text, "02-06-2026");

  console.log(
    `verify-cases.mjs: ${CASES_EXPRESSIONS.length + 3}/${CASES_EXPRESSIONS.length + 3} expressions match CLI --json identically`
  );
  process.exit(0);
} finally {
  fs.rmSync(tmpDir, { recursive: true, force: true });
}
