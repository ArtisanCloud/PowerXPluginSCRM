#!/usr/bin/env node
import fs from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import YAML from "yaml";

const repoRoot = process.cwd();
const workflowsDir = path.join(repoRoot, ".github", "workflows");

async function loadWorkflow(file) {
  const abs = path.join(workflowsDir, file);
  const raw = await fs.readFile(abs, "utf8");
  const doc = YAML.parse(raw) || {};
  return { file, doc };
}

function stepsSignature(doc) {
  const jobs = doc?.jobs || {};
  const result = {};
  for (const [jobName, job] of Object.entries(jobs)) {
    const steps = Array.isArray(job?.steps) ? job.steps : [];
    result[jobName] = steps.map((s) => {
      const wd = s?.["working-directory"] || "";
      const uses = s?.uses || "";
      const run = (s?.run || "").trim().split("\n").map((l) => l.trim()).filter(Boolean)[0] || "";
      return `${wd}::${uses}::${run}`;
    });
  }
  return result;
}

function diffSets(a, b) {
  const onlyA = new Set([...a].filter((x) => !b.has(x)));
  const onlyB = new Set([...b].filter((x) => !a.has(x)));
  return { onlyA, onlyB };
}

const base = await loadWorkflow("ci.yml");
const rel = await loadWorkflow("release.yml");

const aJobs = new Set(Object.keys(base.doc?.jobs || {}));
const bJobs = new Set(Object.keys(rel.doc?.jobs || {}));

console.log(`# CI vs Release Diff\n`);
console.log(`- CI: \`.github/workflows/${base.file}\``);
console.log(`- Release: \`.github/workflows/${rel.file}\`\n`);

{
  const { onlyA, onlyB } = diffSets(aJobs, bJobs);
  if (onlyA.size) console.log(`- Jobs only in CI: ${[...onlyA].sort().join(", ")}`);
  if (onlyB.size) console.log(`- Jobs only in Release: ${[...onlyB].sort().join(", ")}`);
  if (!onlyA.size && !onlyB.size) console.log(`- Jobs: identical`);
  console.log("");
}

const aSig = stepsSignature(base.doc);
const bSig = stepsSignature(rel.doc);

for (const jobName of [...new Set([...Object.keys(aSig), ...Object.keys(bSig)])].sort()) {
  const aSteps = new Set(aSig[jobName] || []);
  const bSteps = new Set(bSig[jobName] || []);
  if (!aSteps.size && !bSteps.size) continue;
  const { onlyA, onlyB } = diffSets(aSteps, bSteps);
  if (!onlyA.size && !onlyB.size) continue;

  console.log(`## job: ${jobName}`);
  if (onlyA.size) {
    console.log(`- Only in CI:`);
    for (const s of [...onlyA].sort()) console.log(`  - ${s}`);
  }
  if (onlyB.size) {
    console.log(`- Only in Release:`);
    for (const s of [...onlyB].sort()) console.log(`  - ${s}`);
  }
  console.log("");
}

