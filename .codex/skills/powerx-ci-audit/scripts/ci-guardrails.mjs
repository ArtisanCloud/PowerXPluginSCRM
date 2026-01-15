#!/usr/bin/env node
import fs from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import YAML from "yaml";

const repoRoot = process.cwd();
const workflowsDir = path.join(repoRoot, ".github", "workflows");

async function exists(p) {
  try {
    await fs.access(p);
    return true;
  } catch {
    return false;
  }
}

function normalizeWorkingDir(step) {
  return step?.["working-directory"] || "";
}

function parseMkdirs(run) {
  const lines = String(run || "")
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean);
  const created = new Set();
  for (const line of lines) {
    const m = line.match(/^mkdir\s+-p\s+(.+)$/);
    if (!m) continue;
    const arg = m[1].trim().replace(/^["']|["']$/g, "");
    if (arg) created.add(arg);
  }
  return created;
}

function inferCreatedDirsFromPxPluginInit(step) {
  const created = new Set();
  const run = String(step?.run || "");
  if (!run.includes("px-plugin") || !run.includes(" init")) return created;

  const wd = normalizeWorkingDir(step);
  const base = wd ? wd : ".";
  const lines = run
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean);

  for (const line of lines) {
    if (!line.includes(" init")) continue;
    // Very small heuristic parser; good enough for CI scripts.
    const tokens = line.split(/\s+/).filter(Boolean);
    const initIdx = tokens.findIndex((t, i) => t === "init" && i > 0);
    if (initIdx < 0) continue;

    let directory = "";
    for (let i = initIdx + 1; i < tokens.length; i++) {
      const t = tokens[i];
      if (t === "--directory" || t === "-directory") {
        directory = tokens[i + 1] || "";
        break;
      }
      if (t.startsWith("--directory=") || t.startsWith("-directory=")) {
        directory = t.split("=", 2)[1] || "";
        break;
      }
    }

    // Find plugin id (first non-flag token after init).
    const pluginIdPattern = /^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$/;
    let pluginId = "";
    for (let i = initIdx + 1; i < tokens.length; i++) {
      const t = tokens[i];
      if (t.startsWith("-")) continue;
      if (pluginIdPattern.test(t) && t.includes(".")) {
        pluginId = t;
        break;
      }
    }
    if (!pluginId) continue;

    // Default target: <working-directory>/<plugin-id>
    // If --directory is provided, treat it as relative to wd unless absolute.
    if (directory) {
      const cleaned = directory.replace(/^["']|["']$/g, "");
      if (cleaned.startsWith("/") || cleaned.match(/^[A-Za-z]:\\/)) {
        // absolute: ignore (outside repo), but still ok for working-dir checks
        continue;
      }
      created.add(path.join(cleaned, "web-admin")); // signal: project root likely exists
      created.add(cleaned);
    } else {
      created.add(path.join(base, pluginId));
    }
  }
  return created;
}

async function loadWorkflow(file) {
  const abs = path.join(workflowsDir, file);
  const raw = await fs.readFile(abs, "utf8");
  const doc = YAML.parse(raw) || {};
  return { file, doc };
}

const release = await loadWorkflow("release.yml");
const jobs = release.doc?.jobs || {};

const problems = [];

for (const [jobName, job] of Object.entries(jobs)) {
  const steps = Array.isArray(job?.steps) ? job.steps : [];

  // Track directories created within this job (best-effort).
  const createdDirs = new Set();
  for (const step of steps) {
    for (const d of parseMkdirs(step?.run)) createdDirs.add(d);
    for (const d of inferCreatedDirsFromPxPluginInit(step)) createdDirs.add(d);
  }

  for (const [idx, step] of steps.entries()) {
    const wd = normalizeWorkingDir(step);
    if (!wd) continue;
    const absWd = path.join(repoRoot, wd);
    const ok = (await exists(absWd)) || createdDirs.has(wd);
    if (!ok) {
      problems.push(
        `release.yml job=${jobName} step#${idx + 1} has working-directory=${wd} but directory does not exist (and not created via mkdir -p in this job)`
      );
    }
  }
}

if (problems.length) {
  console.error(`[ci-guardrails] FAILED (${problems.length})`);
  for (const p of problems) console.error(`- ${p}`);
  process.exit(1);
}

console.log("[ci-guardrails] OK: release.yml working directories look valid");
