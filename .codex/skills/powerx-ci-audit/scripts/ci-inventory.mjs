#!/usr/bin/env node
import fs from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import YAML from "yaml";

const repoRoot = process.cwd();
const workflowsDir = path.join(repoRoot, ".github", "workflows");

async function listWorkflowFiles() {
  const entries = await fs.readdir(workflowsDir, { withFileTypes: true });
  return entries
    .filter((e) => e.isFile() && (e.name.endsWith(".yml") || e.name.endsWith(".yaml")))
    .map((e) => path.join(workflowsDir, e.name))
    .sort();
}

function summarizeOn(onValue) {
  if (!onValue) return "unknown";
  if (typeof onValue === "string") return onValue;
  if (Array.isArray(onValue)) return onValue.join(", ");
  if (typeof onValue === "object") return Object.keys(onValue).sort().join(", ");
  return "unknown";
}

function firstLine(s) {
  if (!s) return "";
  return String(s).split("\n").map((l) => l.trim()).filter(Boolean)[0] ?? "";
}

function normalizeWorkingDir(step) {
  return step?.["working-directory"] || step?.working_directory || "";
}

function asArray(value) {
  if (!value) return [];
  return Array.isArray(value) ? value : [value];
}

function formatCode(code) {
  const trimmed = String(code || "").trim();
  if (!trimmed) return "";
  return "`" + trimmed.replaceAll("`", "\\`") + "`";
}

const files = await listWorkflowFiles();
console.log(`# PowerXPlugin Workflows Inventory\n`);
console.log(`- Repo: ${formatCode(repoRoot)}`);
console.log(`- Workflows: ${files.length}\n`);

for (const filePath of files) {
  const rel = path.relative(repoRoot, filePath);
  const raw = await fs.readFile(filePath, "utf8");
  const doc = YAML.parse(raw);
  const wfName = doc?.name || path.basename(filePath);
  const onSummary = summarizeOn(doc?.on);
  const jobs = doc?.jobs || {};

  console.log(`## ${wfName}`);
  console.log(`- File: ${formatCode(rel)}`);
  console.log(`- Triggers: ${formatCode(onSummary)}`);

  const jobNames = Object.keys(jobs);
  console.log(`- Jobs: ${jobNames.length ? jobNames.join(", ") : "(none)"}\n`);

  for (const jobName of jobNames) {
    const job = jobs[jobName] || {};
    const runsOn = job["runs-on"] ?? job.runs_on ?? "";
    const needs = asArray(job.needs).join(", ");
    const steps = asArray(job.steps);

    console.log(`### job: ${jobName}`);
    if (runsOn) console.log(`- runs-on: ${formatCode(runsOn)}`);
    if (needs) console.log(`- needs: ${formatCode(needs)}`);

    const workingDirs = new Set();
    for (const step of steps) {
      const wd = normalizeWorkingDir(step);
      if (wd) workingDirs.add(wd);
    }
    if (workingDirs.size) {
      console.log(`- working-directories: ${Array.from(workingDirs).sort().map(formatCode).join(" ")}`);
    }

    console.log(`- steps: ${steps.length}`);
    for (const step of steps) {
      const name = step?.name || "";
      const uses = step?.uses || "";
      const run = step?.run || "";
      const wd = normalizeWorkingDir(step);

      const parts = [];
      if (name) parts.push(name);
      if (uses) parts.push(`uses ${uses}`);
      if (run) parts.push(`run ${firstLine(run)}`);
      if (wd) parts.push(`wd=${wd}`);
      if (parts.length) console.log(`  - ${parts.join(" · ")}`);
    }
    console.log("");
  }
}

