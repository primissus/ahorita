import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const publicDir = path.join(__dirname, "public");

await import(path.join(publicDir, "wasm_exec.js"));

const go = new globalThis.Go();
const wasmBytes = fs.readFileSync(path.join(publicDir, "ahorita.wasm"));
const { instance } = await WebAssembly.instantiate(wasmBytes, go.importObject);

const runPromise = go.run(instance);
runPromise.catch((err) => {
  console.error("WASM runtime failed:", err);
  process.exit(1);
});

assert.equal(typeof globalThis.ahoritaRun, "function", "ahoritaRun must be registered on globalThis");

function runExpr(input, optsObjOrStr = "{}") {
  const optsJSON = typeof optsObjOrStr === "string" ? optsObjOrStr : JSON.stringify(optsObjOrStr);
  const raw = globalThis.ahoritaRun(input, optsJSON);
  return { raw, parsed: JSON.parse(raw) };
}

// 1. Required Step 2 smoke check: ahoritaRun("2026-06-02 + 2d", "{}")
const basic = runExpr("2026-06-02 + 2d", "{}");
console.log(basic.raw);
assert.equal(basic.parsed.error, undefined);
assert.equal(basic.parsed.kind, "date");
assert.equal(basic.parsed.text, "2026-06-04");

// 2. Parse error with expected suggestions
const parseErr = runExpr("now +", "{}");
assert.ok(parseErr.parsed.error, "expected parse error for incomplete expression");
assert.equal(parseErr.parsed.error.pos, 5);
assert.ok(Array.isArray(parseErr.parsed.error.expected) && parseErr.parsed.error.expected.length > 0);

// 3. Explicit empty tz must error instead of falling back to UTC
const emptyTz = runExpr("now", { tz: "" });
assert.ok(emptyTz.parsed.error?.message.includes("invalid tz"));

// 4. Spanish input + options override + explain steps
const spanish = runExpr("hace 2 días", {
  now: "2026-11-14T14:30:00-06:00",
  tz: "America/Mexico_City",
  type: "relative",
  lang: "es",
  explain: true,
});
assert.equal(spanish.parsed.error, undefined);
assert.equal(spanish.parsed.text, "hace 2 días");
assert.equal(spanish.parsed.explain?.length, 1);

// 5. Plan 9 business days (MX Revolution Day Nov 16 skipped)
const biz = runExpr("2026-11-13 + 3 business days", {
  now: "2026-11-14T14:30:00-06:00",
  tz: "America/Mexico_City",
});
assert.equal(biz.parsed.error, undefined);
assert.equal(biz.parsed.text, "2026-11-19");

// 6. Plan 10 recurrence + cron + cross-zone weekday shift note
const cronNote = runExpr("every monday 8pm, to UTC, to cron", {
  now: "2026-11-14T14:30:00-06:00",
  tz: "America/Mexico_City",
});
assert.equal(cronNote.parsed.error, undefined);
assert.equal(cronNote.parsed.text, "0 2 * * 2");
assert.ok(cronNote.parsed.note?.includes("weekday shifts from Monday to Tuesday in UTC"));

console.log("smoke.mjs: all assertions passed");
process.exit(0);
