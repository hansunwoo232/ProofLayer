#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
WORKSPACE_ROOT="$(cd "${PROJECT_ROOT}/../.." && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"
EVIDENCE_DIR="${WORKSPACE_ROOT}/work/prooflayer-lab/evidence"
CORRELATION_ID="${1:-}"

if [[ ! "${CORRELATION_ID}" =~ ^PL-[A-F0-9]{32}$ ]]; then
  echo "Usage: $0 PL-<32 uppercase hexadecimal characters>" >&2
  exit 2
fi

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Missing ${ENV_FILE}. Run ./setup.sh first." >&2
  exit 1
fi

set -a
source "${ENV_FILE}"
set +a

if [[ -z "${PROOFLAYER_OBSERVER_PASSWORD:-}" ]]; then
  echo "Missing observer credential. Run ./configure-observer-role.sh first." >&2
  exit 1
fi

search="search index=prooflayer_test source=\"prooflayer:windows-lab\" earliest=-24h latest=now \"${CORRELATION_ID}\" | spath | eval correlation_id=mvindex(correlation_id, 0), provider=mvindex(provider, 0), event_id=mvindex(event_id, 0), record_id=mvindex(record_id, 0), endpoint_event_time=mvindex(endpoint_event_time, 0), host_name=mvindex('host.name', 0), process_name=mvindex('process.name', 0), process_command_line=mvindex('process.command_line', 0), user_name=mvindex('user.name', 0) | where correlation_id=\"${CORRELATION_ID}\" AND event_id=1 | eval required_field_count=4, missing_field_count=if(isnull(host_name),1,0)+if(isnull(process_name),1,0)+if(isnull(process_command_line),1,0)+if(isnull(user_name),1,0), matched_fields=if(missing_field_count=0,\"host.name,process.name,process.command_line,user.name\",null()), ingestion_latency_ms=round((_indextime-_time)*1000, 0) | eventstats count as match_count | where match_count=1 AND missing_field_count=0 | head 1 | table correlation_id, provider, event_id, record_id, endpoint_event_time, ingestion_latency_ms, match_count, required_field_count, missing_field_count, matched_fields"

result="$(curl -ksS \
  --max-time 20 \
  -u "prooflayer_observer:${PROOFLAYER_OBSERVER_PASSWORD}" \
  https://127.0.0.1:8089/services/search/jobs/export \
  --data-urlencode "search=${search}" \
  --data "output_mode=json")"

if ! grep -Eq '"correlation_id"[[:space:]]*:[[:space:]]*"'"${CORRELATION_ID}"'"' <<< "${result}"; then
  echo "The exact correlation event was missing, ambiguous, or lacked required fields." >&2
  exit 1
fi

if ! grep -Eq '"match_count"[[:space:]]*:[[:space:]]*"?1"?' <<< "${result}"; then
  echo "Expected exactly one correlation match." >&2
  exit 1
fi

mkdir -p "${EVIDENCE_DIR}"
EVIDENCE_FILE="${EVIDENCE_DIR}/day-12-${CORRELATION_ID}.json"
umask 077
printf '%s\n' "${result}" > "${EVIDENCE_FILE}"

echo "PASS: exact Windows correlation event and required field presence verified."
echo "Evidence: ${EVIDENCE_FILE}"
printf '%s\n' "${result}"
