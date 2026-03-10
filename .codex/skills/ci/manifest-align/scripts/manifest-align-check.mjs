#!/usr/bin/env node

import { execSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const args = new Set(process.argv.slice(2));
const autoFix = args.has("--fix");
const autoStage = args.has("--stage");

function run(command, options = {}) {
  return execSync(command, {
    cwd: repoRoot,
    stdio: ["ignore", "pipe", "pipe"],
    encoding: "utf8",
    ...options,
  });
}
function fail(message) { console.error(`❌ ${message}`); process.exit(1); }
function info(message) { console.log(`ℹ️  ${message}`); }
function ok(message) { console.log(`✅ ${message}`); }
function listCapabilityFiles() {
  const dir = path.join(repoRoot, "contracts", "capabilities");
  if (!fs.existsSync(dir)) return [];
  return fs.readdirSync(dir).filter((name) => name.endsWith(".yaml")).sort().map((name) => path.join(dir, name));
}
function parseCapability(content, filePath) {
  const idMatch = content.match(/^id:\s*([^\n]+)$/m);
  if (!idMatch) fail(`capability 缺少 id: ${filePath}`);
  const id = idMatch[1].trim();
  const rbacBlock = content.match(/rbac:\s*([\s\S]*?)(?:\n[a-zA-Z_][^:\n]*:|\n$)/m);
  if (!rbacBlock) fail(`capability 缺少 rbac 段: ${filePath}`);
  const resourceMatch = rbacBlock[1].match(/resource:\s*([^\n]+)$/m);
  let actions = [];
  const actionsInlineMatch = rbacBlock[1].match(/actions:\s*\[([^\]]+)\]/m);
  if (actionsInlineMatch) actions = actionsInlineMatch[1].split(",").map((item) => item.trim()).filter(Boolean);
  if (!resourceMatch || actions.length === 0) fail(`capability rbac 缺少 resource/actions: ${filePath}`);
  return { id, resource: resourceMatch[1].trim(), actions, firstAction: actions[0] };
}
function parseExposure(content) {
  const channels = [];
  const lines = content.split("\n");
  let current = null;
  for (const line of lines) {
    const cap = line.match(/^\s*-?\s*capability:\s*(\S+)\s*$/);
    if (cap) { current = { capability: cap[1], rbac: "" }; channels.push(current); continue; }
    const rbac = line.match(/^\s*-?\s*rbac:\s*(\S+)\s*$/);
    if (rbac && current) current.rbac = rbac[1];
  }
  return channels;
}
function parseRBACResources(content) {
  const resources = new Map();
  let current = "";
  let collecting = false;
  for (const line of content.split("\n")) {
    const resource = line.match(/^\s*-?\s*resource:\s*(\S+)\s*$/);
    if (resource) { current = resource[1]; if (!resources.has(current)) resources.set(current, new Set()); collecting = false; continue; }
    const inline = line.match(/^\s*-?\s*actions:\s*\[([^\]]+)\]\s*$/);
    if (inline && current) { inline[1].split(",").map((item) => item.trim()).filter(Boolean).forEach((action) => resources.get(current).add(action)); collecting = false; continue; }
    if (/^\s*-?\s*actions:\s*$/.test(line)) { collecting = true; continue; }
    const action = line.match(/^\s*-\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*$/);
    if (collecting && action && current) resources.get(current).add(action[1]);
  }
  return resources;
}
function listDriftFiles(targets) {
  const output = run(`git diff --name-only -- ${targets.join(" ")}`).trim();
  if (!output) return [];
  return output.split("\n").map((item) => item.trim()).filter(Boolean);
}
info("执行 plugin catalog 重建检查…");
try { process.stdout.write(run("make plugin-yaml-check")); } catch (error) { process.stdout.write(error.stdout || ""); process.stderr.write(error.stderr || ""); fail("make plugin-yaml-check 执行失败"); }
const driftTargets = ["plugin.d/capabilities.yaml", "plugin.d/exposure.yaml", "plugin.d/rbac.yaml"];
const driftFiles = listDriftFiles(driftTargets);
if (driftFiles.length > 0 && !autoFix) fail(`检测到 catalog 漂移（请提交同步结果）:\n${driftFiles.join("\n")}`);
if (driftFiles.length > 0 && autoFix) {
  ok(`已自动同步 catalog：\n${driftFiles.join("\n")}`);
  if (autoStage) { run(`git add -- ${driftFiles.join(" ")}`); ok("已自动 git add 同步产物"); }
}
if (driftFiles.length === 0) ok("plugin.d 产物无漂移");
const capabilities = listCapabilityFiles().map((filePath) => parseCapability(fs.readFileSync(filePath, "utf8"), filePath));
const exposureChannels = parseExposure(fs.readFileSync(path.join(repoRoot, "plugin.d", "exposure.yaml"), "utf8"));
for (const capability of capabilities) {
  const matched = exposureChannels.find((item) => item.capability === capability.id);
  if (!matched) fail(`exposure 缺少 capability: ${capability.id}`);
  if (matched.rbac && matched.rbac !== `${capability.resource}:${capability.firstAction}`) fail(`exposure rbac 不匹配: ${capability.id}`);
}
ok("capability -> exposure.rbac 映射通过");
const rbacResources = parseRBACResources(fs.readFileSync(path.join(repoRoot, "plugin.d", "rbac.yaml"), "utf8"));
for (const capability of capabilities) {
  const actions = rbacResources.get(capability.resource);
  if (!actions) fail(`rbac.resources 缺少 resource: ${capability.resource}`);
  for (const action of capability.actions) if (!actions.has(action)) fail(`rbac.resources 缺少 action: resource=${capability.resource}, action=${action}`);
}
ok("capability -> rbac.resources 覆盖通过");
ok("Manifest 对齐检查全部通过");
