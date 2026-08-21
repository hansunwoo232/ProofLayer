import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import path from "node:path";

const toolDirectory = path.dirname(fileURLToPath(import.meta.url));
const wireframePath = path.resolve(toolDirectory, "..", "dashboard", "result-screen-wireframe.html");
const source = await readFile(wireframePath, "utf8");
const loginPath = path.resolve(toolDirectory, "..", "dashboard", "login.html");
const loginSource = await readFile(loginPath, "utf8");

assert.match(source, /<html lang="en">/);
assert.match(source, /<main id="result">/);
assert.match(source, /aria-label="Primary navigation"/);
assert.match(source, />\s*PASS</);
assert.match(source, />\s*FAIL</);
assert.match(source, />\s*NOT TESTED</);
assert.match(source, /Evidence boundary:/);
assert.match(source, /id="run-test"[^>]*disabled/);
assert.match(source, /role="status" aria-live="polite"/);
assert.match(source, /fetch\("\/v1\/session"/);
assert.match(source, /fetch\("\/v1\/test-jobs"/);
assert.match(source, /fetch\(`\/v1\/test-jobs\/\$\{encodeURIComponent\(jobId\)\}`/);
assert.match(source, /"Idempotency-Key": activeIdempotencyKey/);
assert.match(source, /"X-ProofLayer-CSRF": csrfToken/);
assert.match(source, /scenario_id: "windows-process-marker"/);
assert.match(source, /const maximumStatusPolls = 150/);
assert.match(source, /endpoint_event_delayed/);
assert.match(source, /The endpoint event is taking longer than expected\./);
assert.doesNotMatch(source, /setInterval\(/);
assert.doesNotMatch(source, /\.innerHTML\s*=/);
assert.doesNotMatch(source, /\b(?:command|arguments):\s*["'`]/);
assert.doesNotMatch(source, /<script[^>]+src=/);
assert.doesNotMatch(source, /https?:\/\/[^\s"']+\.(?:js|css)/i);
assert.match(source, /session\.authenticated === false/);
assert.match(source, /fetch\("\/v1\/auth\/logout"/);

assert.match(loginSource, /<html lang="en">/);
assert.match(loginSource, /autocomplete="username"/);
assert.match(loginSource, /autocomplete="current-password"/);
assert.match(loginSource, /fetch\("\/v1\/session"/);
assert.match(loginSource, /fetch\("\/v1\/auth\/login"/);
assert.match(loginSource, /"X-ProofLayer-CSRF": csrfToken/);
assert.match(loginSource, /credentials: "same-origin"/);
assert.doesNotMatch(loginSource, /localStorage|sessionStorage|\.innerHTML\s*=/);
assert.doesNotMatch(loginSource, /<script[^>]+src=/);
assert.doesNotMatch(loginSource, /https?:\/\/[^\s"']+\.(?:js|css)/i);

const clickHandlerStart = source.indexOf('button.addEventListener("click"');
const disableBeforePost = source.indexOf("button.disabled = true", clickHandlerStart);
const jobPost = source.indexOf('fetch("/v1/test-jobs"', clickHandlerStart);
assert.ok(clickHandlerStart > 0 && disableBeforePost > clickHandlerStart && disableBeforePost < jobPost,
  "Run Test must disable synchronously before the queue request");

const stageNames = [...source.matchAll(/<li class="stage" data-stage="([a-z_]+)">/g)].map((match) => match[1]);
assert.deepEqual(stageNames, [
  "execution",
  "endpoint_telemetry",
  "siem_ingestion",
  "field_validation",
  "detection",
  "alert",
  "cleanup",
]);

const dataMatch = source.match(
  /<script id="prooflayer-run" type="application\/json">\s*([\s\S]*?)\s*<\/script>/,
);
assert.ok(dataMatch, "wireframe must embed bounded representative result data");
const result = JSON.parse(dataMatch[1]);

assert.equal(result.schema_version, "1.0");
assert.equal(result.overall_status, "failed");
assert.deepEqual(result.stage_statuses, [
  "passed",
  "passed",
  "passed",
  "failed",
  "not_tested",
  "not_tested",
  "passed",
]);
assert.equal(result.raw_event_included, false);

console.log("PASS local authenticated dashboard contract");
