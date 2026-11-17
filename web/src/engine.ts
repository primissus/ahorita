export interface ResultStep {
  stage: string;
  value: string;
}

export interface ExprError {
  pos: number;
  message: string;
  expected?: string[];
}

export type ResultKind =
  | "date"
  | "datetime"
  | "time"
  | "duration"
  | "span"
  | "number"
  | "list"
  | "text"
  | "recurrence";

export interface Result {
  input: string;
  alias?: string;
  kind?: ResultKind | "";
  text?: string;
  iso?: string;
  unix?: number;
  start?: string;
  end?: string;
  note?: string;
  explain?: ResultStep[];
  error?: ExprError;
}

export type OutputType = "" | "date" | "datetime" | "time" | "range" | "relative";
export type Language = "en" | "es";
export type WeekOneRule = "iso" | "first-day" | "full" | "days";
export type MonthOverflowRule = "clamp" | "overflow";

export interface RunOptions {
  now?: string;
  tz?: string;
  type?: OutputType;
  format?: string;
  explain?: boolean;
  lang?: Language;
  strict?: boolean;
  week_start?: string;
  week_one?: WeekOneRule;
  month_overflow?: MonthOverflowRule;
  count_inclusive?: boolean;
  country?: string;
  weekend?: string | string[];
  days_off?: string[];
  aliases?: Record<string, string>;
}

export interface BrowserTimeInfo {
  now: string;
  tz: string;
  tzFallbackUsed: boolean;
}

interface GoConstructor {
  new (): {
    importObject: WebAssembly.Imports;
    run(instance: WebAssembly.Instance): Promise<void>;
  };
}

declare global {
  var Go: GoConstructor | undefined;
  var ahoritaRun: ((input: string, optionsJSON: string) => string) | undefined;
}

let initPromise: Promise<void> | null = null;

/**
 * Single browser-clock/zone boundary for the GUI: all date/time computation
 * happens in the Go WASM engine (`expr.Run`).
 */
export function readCurrentTimeAndZone(): BrowserTimeInfo {
  const now = new Date().toISOString();
  let resolvedTZ = "";
  try {
    resolvedTZ = Intl.DateTimeFormat().resolvedOptions().timeZone ?? "";
  } catch {
    resolvedTZ = "";
  }
  const trimmed = resolvedTZ.trim();
  if (!trimmed || trimmed.toLowerCase() === "local") {
    return { now, tz: "UTC", tzFallbackUsed: true };
  }
  return { now, tz: trimmed, tzFallbackUsed: false };
}

async function instantiateInNode(): Promise<WebAssembly.Instance> {
  const fsMod = "node:" + "fs";
  const pathMod = "node:" + "path";
  const urlMod = "node:" + "url";
  const fs = (await import(/* @vite-ignore */ fsMod)) as typeof import("node:fs");
  const path = (await import(/* @vite-ignore */ pathMod)) as typeof import("node:path");
  const { fileURLToPath, pathToFileURL } = (await import(
    /* @vite-ignore */ urlMod
  )) as typeof import("node:url");

  const here = path.dirname(fileURLToPath(import.meta.url));
  const publicDir = path.resolve(here, "../public");
  await import(/* @vite-ignore */ pathToFileURL(path.join(publicDir, "wasm_exec.js")).href);

  const GoCtor = globalThis.Go;
  if (!GoCtor) {
    throw new Error("wasm_exec.js did not define globalThis.Go");
  }
  const go = new GoCtor();
  const wasmBytes = fs.readFileSync(path.join(publicDir, "ahorita.wasm"));
  const { instance } = await WebAssembly.instantiate(wasmBytes, go.importObject);
  void go.run(instance).catch((err) => {
    console.error("ahorita WASM runtime exited unexpectedly:", err);
  });
  return instance;
}

async function instantiateInBrowser(): Promise<WebAssembly.Instance> {
  const GoCtor = globalThis.Go;
  if (!GoCtor) {
    throw new Error("wasm_exec.js is not loaded (globalThis.Go missing)");
  }
  const go = new GoCtor();
  const baseUrl = import.meta.env.BASE_URL || "/";
  const wasmUrl = `${baseUrl.replace(/\/$/, "")}/ahorita.wasm`;

  let instantiated: WebAssembly.WebAssemblyInstantiatedSource;
  if (typeof WebAssembly.instantiateStreaming === "function") {
    const response = await fetch(wasmUrl);
    if (!response.ok) {
      throw new Error(`Failed to fetch ${wasmUrl}: ${response.status} ${response.statusText}`);
    }
    try {
      instantiated = await WebAssembly.instantiateStreaming(response.clone(), go.importObject);
    } catch {
      const bytes = await response.arrayBuffer();
      instantiated = await WebAssembly.instantiate(bytes, go.importObject);
    }
  } else {
    const response = await fetch(wasmUrl);
    if (!response.ok) {
      throw new Error(`Failed to fetch ${wasmUrl}: ${response.status} ${response.statusText}`);
    }
    const bytes = await response.arrayBuffer();
    instantiated = await WebAssembly.instantiate(bytes, go.importObject);
  }

  void go.run(instantiated.instance).catch((err) => {
    console.error("ahorita WASM runtime exited unexpectedly:", err);
  });
  return instantiated.instance;
}

export function initEngine(): Promise<void> {
  if (typeof globalThis.ahoritaRun === "function") {
    return Promise.resolve();
  }
  if (!initPromise) {
    initPromise = (async () => {
      const isNode =
        typeof process !== "undefined" && Boolean(process.versions?.node);
      if (isNode) {
        await instantiateInNode();
      } else {
        await instantiateInBrowser();
      }
      if (typeof globalThis.ahoritaRun !== "function") {
        throw new Error("ahoritaRun was not registered by WASM module");
      }
    })();
    initPromise.catch(() => {
      initPromise = null;
    });
  }
  return initPromise;
}

export function run(input: string, options: RunOptions = {}): Result {
  if (typeof globalThis.ahoritaRun !== "function") {
    throw new Error("Engine not initialized; call initEngine() before run()");
  }

  let payload: RunOptions = options;
  const needsNow = options.now === undefined;
  const needsTZ = options.tz === undefined;
  if (needsNow || needsTZ) {
    const current = readCurrentTimeAndZone();
    payload = {
      ...options,
      now: needsNow ? current.now : options.now,
      tz: needsTZ ? current.tz : options.tz,
    };
  }

  const raw = globalThis.ahoritaRun(input, JSON.stringify(payload));
  return JSON.parse(raw) as Result;
}
