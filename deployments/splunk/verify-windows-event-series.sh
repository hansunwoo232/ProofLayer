#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
WORKSPACE_ROOT="$(cd "${PROJECT_ROOT}/../.." && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"
EVIDENCE_DIR="${WORKSPACE_ROOT}/work/prooflayer-lab/evidence"
EVIDENCE_FILE="${EVIDENCE_DIR}/day-07-splunk-series.json"
EXPECTED_SAMPLE_COUNT=3

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

search='search index=prooflayer_test source="prooflayer:windows-lab" | spath | eval correlation_id=mvindex(correlation_id, 0), provider=mvindex(provider, 0), event_id=mvindex(event_id, 0), record_id=mvindex(record_id, 0), endpoint_event_time=mvindex(endpoint_event_time, 0), host_name=mvindex('\''host.name'\'', 0), process_name=mvindex('\''process.name'\'', 0), process_command_line=mvindex('\''process.command_line'\'', 0), user_name=mvindex('\''user.name'\'', 0) | eval missing_field_count=if(isnull(host_name),1,0)+if(isnull(process_name),1,0)+if(isnull(process_command_line),1,0)+if(isnull(user_name),1,0) | where match(correlation_id, "^PL-[A-F0-9]{32}$") AND event_id=1 AND missing_field_count=0 | eval ingestion_latency_ms=round((_indextime-_time)*1000, 0), matched_fields="host.name,process.name,process.command_line,user.name" | sort 0 - _indextime | head 3 | eventstats count as sample_count min(ingestion_latency_ms) as min_ingestion_latency_ms avg(ingestion_latency_ms) as average_ingestion_latency_ms max(ingestion_latency_ms) as max_ingestion_latency_ms | eval average_ingestion_latency_ms=round(average_ingestion_latency_ms, 0) | table correlation_id, event_id, record_id, endpoint_event_time, ingestion_latency_ms, missing_field_count, matched_fields, sample_count, min_ingestion_latency_ms, average_ingestion_latency_ms, max_ingestion_latency_ms'

result="$(curl -ksS \
  -u "prooflayer_observer:${PROOFLAYER_OBSERVER_PASSWORD}" \
  https://127.0.0.1:8089/services/search/jobs/export \
  --data-urlencode "search=${search}" \
  --data "output_mode=json")"

sample_count="$(grep -c '"correlation_id"' <<< "${result}" || true)"
if [[ "${sample_count}" -ne "${EXPECTED_SAMPLE_COUNT}" ]]; then
  echo "Expected ${EXPECTED_SAMPLE_COUNT} canonical Windows events, received ${sample_count}." >&2
  exit 1
fi

unique_count="$(grep -o 'PL-[A-F0-9]\{32\}' <<< "${result}" | sort -u | wc -l | tr -d ' ')"
if [[ "${unique_count}" -ne "${EXPECTED_SAMPLE_COUNT}" ]]; then
  echo "The Windows proof series does not contain ${EXPECTED_SAMPLE_COUNT} unique correlation IDs." >&2
  exit 1
fi

mkdir -p "${EVIDENCE_DIR}"
umask 077
printf '%s\n' "${result}" > "${EVIDENCE_FILE}"

echo "PASS: ${EXPECTED_SAMPLE_COUNT} unique Windows Sysmon events verified in prooflayer_test."
echo "Evidence: ${EVIDENCE_FILE}"
printf '%s\n' "${result}"
