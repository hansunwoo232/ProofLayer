#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
WORKSPACE_ROOT="$(cd "${PROJECT_ROOT}/../.." && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"
EVIDENCE_DIR="${WORKSPACE_ROOT}/work/prooflayer-lab/evidence"
EVIDENCE_FILE="${EVIDENCE_DIR}/day-07-latest-splunk-result.json"

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

search='search index=prooflayer_test source="prooflayer:windows-lab" | spath | eval correlation_id=mvindex(correlation_id, 0), provider=mvindex(provider, 0), event_id=mvindex(event_id, 0), record_id=mvindex(record_id, 0), endpoint_event_time=mvindex(endpoint_event_time, 0), host_name=mvindex('\''host.name'\'', 0), process_name=mvindex('\''process.name'\'', 0), process_command_line=mvindex('\''process.command_line'\'', 0), user_name=mvindex('\''user.name'\'', 0) | eval missing_field_count=if(isnull(host_name),1,0)+if(isnull(process_name),1,0)+if(isnull(process_command_line),1,0)+if(isnull(user_name),1,0) | where match(correlation_id, "^PL-[A-F0-9]{32}$") AND missing_field_count=0 | eval ingestion_latency_ms=round((_indextime-_time)*1000, 0), matched_fields="host.name,process.name,process.command_line,user.name" | sort 0 - _indextime | head 1 | table correlation_id, provider, event_id, record_id, endpoint_event_time, ingestion_latency_ms, missing_field_count, matched_fields'

result="$(curl -ksS \
  -u "prooflayer_observer:${PROOFLAYER_OBSERVER_PASSWORD}" \
  https://127.0.0.1:8089/services/search/jobs/export \
  --data-urlencode "search=${search}" \
  --data "output_mode=json")"

if ! grep -Eq '"correlation_id"[[:space:]]*:[[:space:]]*"PL-[A-F0-9]{32}"' <<< "${result}"; then
  echo "No canonical Windows correlation event was returned by Splunk." >&2
  exit 1
fi

if ! grep -Eq '"event_id"[[:space:]]*:[[:space:]]*"?1"?' <<< "${result}"; then
  echo "The latest Windows event is not Sysmon Event ID 1." >&2
  exit 1
fi

mkdir -p "${EVIDENCE_DIR}"
umask 077
printf '%s\n' "${result}" > "${EVIDENCE_FILE}"

echo "PASS: canonical Windows Sysmon event found in prooflayer_test."
echo "Evidence: ${EVIDENCE_FILE}"
printf '%s\n' "${result}"
