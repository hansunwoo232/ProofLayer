import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import path from "node:path";

const toolDirectory = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(toolDirectory, "..", "schemas", "v1");

async function readJson(relativePath) {
  return JSON.parse(await readFile(path.join(root, relativePath), "utf8"));
}

const scenario = await readJson("examples/scenario-process-marker.json");
const registryScenario = await readJson("examples/scenario-registry-run-key-canary.json");
const scheduledTaskScenario = await readJson("examples/scenario-scheduled-task-canary.json");
const runnerResults = await Promise.all([
  readJson("examples/runner-result-process-marker.json"),
  readJson("examples/runner-result-registry-canary.json"),
  readJson("examples/runner-result-scheduled-task-canary.json"),
]);
const job = await readJson("examples/test-job.json");
const run = await readJson("examples/test-run-parser-failure.json");

const uuidPattern =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const versionPattern = /^\d+\.\d+\.\d+$/;
const scenarioIdPattern = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const correlationPattern = /^PL-[A-F0-9]{32}$/;
const scenarioTopLevelFields = new Set([
  "schema_version",
  "scenario_id",
  "scenario_version",
  "name",
  "description",
  "platform",
  "attack_technique",
  "action",
  "execution",
  "risk_level",
  "timeout_seconds",
  "cleanup_required",
  "cleanup",
  "parameter_schema",
  "expected_telemetry",
]);
const prohibitedExecutionFieldPattern =
  /^(?:command|cmd|shell|script|arguments|args|executable|executable_path|url|payload)$/i;

const allowedActions = new Set([
  "emit_process_marker",
  "create_registry_canary",
  "create_scheduled_task_canary",
  "create_local_user_canary",
  "create_service_canary",
  "query_dns_canary",
]);

const executionPolicyByAction = new Map([
  ["emit_process_marker", ["builtin.emit_process_marker", "none", "verify_no_artifacts"]],
  ["create_registry_canary", ["builtin.create_registry_canary", "none", "remove_registry_value"]],
  ["create_scheduled_task_canary", ["builtin.create_scheduled_task_canary", "none", "delete_scheduled_task"]],
  ["create_local_user_canary", ["builtin.create_local_user_canary", "none", "delete_local_user"]],
  ["create_service_canary", ["builtin.create_service_canary", "none", "delete_service"]],
  ["query_dns_canary", ["builtin.query_dns_canary", "dns_only", "verify_no_artifacts"]],
]);

function validateNoArbitraryExecutionFields(value, objectPath = "scenario") {
  if (Array.isArray(value)) {
    value.forEach((item, index) =>
      validateNoArbitraryExecutionFields(item, `${objectPath}[${index}]`),
    );
    return;
  }

  if (value === null || typeof value !== "object") {
    return;
  }

  for (const [key, child] of Object.entries(value)) {
    assert.ok(
      !prohibitedExecutionFieldPattern.test(key),
      `Arbitrary execution field is prohibited at ${objectPath}.${key}`,
    );
    validateNoArbitraryExecutionFields(child, `${objectPath}.${key}`);
  }
}

function validateScenarioSafety(candidate) {
  for (const key of Object.keys(candidate)) {
    assert.ok(scenarioTopLevelFields.has(key), `Unknown scenario field: ${key}`);
  }
  validateNoArbitraryExecutionFields(candidate);
  assert.ok(allowedActions.has(candidate.action), "Scenario action must be allowlisted");
  assert.equal(candidate.cleanup_required, true, "Scenario cleanup must be required");
  assert.equal(candidate.cleanup.verify_absence, true, "Cleanup must verify artifact absence");
  assert.ok(candidate.cleanup.max_attempts >= 1 && candidate.cleanup.max_attempts <= 3);

  const [handler, networkAccess, cleanupStrategy] = executionPolicyByAction.get(candidate.action);
  assert.equal(candidate.execution.handler, handler, "Action must use its fixed built-in handler");
  assert.equal(candidate.execution.network_access, networkAccess, "Network policy must match the action");
  assert.equal(candidate.cleanup.strategy, cleanupStrategy, "Cleanup strategy must match the action");
}

assert.equal(scenario.schema_version, "1.0");
assert.match(scenario.scenario_id, scenarioIdPattern);
assert.match(scenario.scenario_version, versionPattern);
validateScenarioSafety(scenario);
validateScenarioSafety(registryScenario);
validateScenarioSafety(scheduledTaskScenario);

const runnerResultFields = [
  "cleanup_status",
  "completed_at",
  "correlation_id",
  "latency_ms",
  "scenario_id",
  "scenario_version",
  "schema_version",
  "started_at",
  "status",
];
for (const result of runnerResults) {
  assert.deepEqual(
    Object.keys(result).sort(),
    runnerResultFields,
    `${result.scenario_id} must use the canonical Runner result shape`,
  );
  assert.equal(result.schema_version, "1.0");
  assert.equal(result.status, "passed");
  assert.equal(result.cleanup_status, "passed");
  assert.match(result.correlation_id, correlationPattern);
  assert.match(result.scenario_id, scenarioIdPattern);
  assert.match(result.scenario_version, versionPattern);
  assert.ok(Date.parse(result.started_at) <= Date.parse(result.completed_at));
  assert.ok(Number.isInteger(result.latency_ms) && result.latency_ms >= 0);
}
assert.equal(
  new Set(runnerResults.map(({ scenario_id }) => scenario_id)).size,
  3,
  "Runner result comparison must cover three scenarios",
);
assert.ok(
  Number.isInteger(scenario.timeout_seconds) &&
    scenario.timeout_seconds >= 1 &&
    scenario.timeout_seconds <= 300,
  "Scenario timeout must be between 1 and 300 seconds",
);
assert.equal(scenario.cleanup_required, true);
assert.ok(scenario.expected_telemetry.event_ids.length > 0);
assert.ok(scenario.expected_telemetry.required_fields.length > 0);

assert.equal(job.schema_version, "1.0");
for (const id of [
  job.job_id,
  job.environment_id,
  job.host_id,
  job.requested_by,
]) {
  assert.match(id, uuidPattern);
}
assert.match(job.correlation_id, correlationPattern);
assert.match(job.scenario_id, scenarioIdPattern);
assert.match(job.scenario_version, versionPattern);
assert.ok(Date.parse(job.requested_at) < Date.parse(job.expires_at));
assert.match(job.nonce, /^[A-Za-z0-9_-]{22,64}$/);
assert.equal(job.signature.algorithm, "Ed25519");
assert.match(job.signature.value, /^[A-Za-z0-9_-]{80,120}$/);

assert.equal(run.schema_version, "1.0");
assert.match(run.run_id, uuidPattern);
assert.equal(run.job_id, job.job_id);
assert.equal(run.correlation_id, job.correlation_id);
assert.equal(run.environment_id, job.environment_id);
assert.equal(run.host.host_id, job.host_id);
assert.equal(run.scenario.scenario_id, scenario.scenario_id);
assert.equal(run.scenario.scenario_version, scenario.scenario_version);

const expectedStageOrder = [
  "execution",
  "endpoint_telemetry",
  "siem_ingestion",
  "field_validation",
  "detection",
  "alert",
  "cleanup",
];
assert.deepEqual(
  run.stages.map(({ stage }) => stage),
  expectedStageOrder,
  "Stage order must be deterministic",
);
assert.equal(new Set(expectedStageOrder).size, run.stages.length);

const failedStage = run.stages.find(({ status }) => status === "failed");
assert.ok(failedStage, "A failed run must include a failed stage");
assert.equal(run.overall_status, "failed");
assert.ok(run.root_cause, "A failed run must include a root cause");

const cleanup = run.stages.find(({ stage }) => stage === "cleanup");
assert.equal(cleanup.required, true);
assert.equal(cleanup.status, "passed");

for (const stage of run.stages) {
  assert.ok(
    !("raw_event" in stage.evidence),
    "Default evidence must not include raw customer events",
  );
}

console.log("PASS scenario example");
console.log("PASS registry canary scenario example");
console.log("PASS scheduled task canary scenario example");
console.log("PASS three versioned Runner result examples share one shape");
console.log("PASS test job example");
console.log("PASS parser-failure run example");
console.log("PASS cross-contract security invariants");

for (const unsafeMutation of [
  { action: "run_arbitrary_command" },
  { command: "whoami" },
  { execution: { ...scenario.execution, arguments: ["/c", "whoami"] } },
  { unsupported_property: true },
  { cleanup_required: false },
  { execution: { ...scenario.execution, handler: "builtin.query_dns_canary" } },
  { cleanup: { ...scenario.cleanup, verify_absence: false } },
]) {
  const unsafeScenario = structuredClone(scenario);
  Object.assign(unsafeScenario, unsafeMutation);
  assert.throws(() => validateScenarioSafety(unsafeScenario));
}

console.log("PASS unsafe scenario mutations rejected");
