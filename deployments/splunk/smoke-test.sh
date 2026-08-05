#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"
INDEX_NAME="prooflayer_test"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Missing ${ENV_FILE}. Run ./setup.sh first." >&2
  exit 1
fi

set -a
source "${ENV_FILE}"
set +a

event="prooflayer_day5_smoke=true source=prooflayer:day5"
status_code="$(curl -ksS -o /dev/null -w '%{http_code}' \
  -u "admin:${SPLUNK_PASSWORD}" \
  --data-binary "${event}" \
  "https://127.0.0.1:8089/services/receivers/simple?index=${INDEX_NAME}&sourcetype=prooflayer:lab")"

if [[ "${status_code}" != "200" ]]; then
  echo "Splunk ingest returned HTTP ${status_code}." >&2
  exit 1
fi

sleep 3

result="$(curl -ksS \
  -u "admin:${SPLUNK_PASSWORD}" \
  "https://127.0.0.1:8089/services/search/jobs/export" \
  --data-urlencode "search=search index=${INDEX_NAME} prooflayer_day5_smoke=true | stats count" \
  --data "output_mode=json")"

if ! grep -Eq '"count"[[:space:]]*:[[:space:]]*"[1-9][0-9]*"' <<< "${result}"; then
  echo "The smoke event was ingested but was not returned by search." >&2
  exit 1
fi

echo "PASS: a synthetic event was ingested into ${INDEX_NAME} and returned by search."
