#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"
INPUT_NAME="prooflayer_lab"
INDEX_NAME="prooflayer_test"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Missing ${ENV_FILE}. Run ./setup.sh first." >&2
  exit 1
fi

set -a
source "${ENV_FILE}"
set +a

if [[ -z "${PROOFLAYER_HEC_TOKEN:-}" || ${#PROOFLAYER_HEC_TOKEN} -lt 32 ]]; then
  PROOFLAYER_HEC_TOKEN="$(openssl rand -hex 32)"
  temporary_env="$(mktemp "${SCRIPT_DIR}/.env.tmp.XXXXXX")"
  chmod 600 "${temporary_env}"
  grep -v '^PROOFLAYER_HEC_TOKEN=' "${ENV_FILE}" > "${temporary_env}" || true
  printf 'PROOFLAYER_HEC_TOKEN=%s\n' "${PROOFLAYER_HEC_TOKEN}" >> "${temporary_env}"
  mv "${temporary_env}" "${ENV_FILE}"
  echo "Generated and stored an ignored local HEC token."
fi

input_url="https://127.0.0.1:8089/services/data/inputs/http/${INPUT_NAME}"
existing_code="$(curl -ksS -o /dev/null -w '%{http_code}' \
  -u "admin:${SPLUNK_PASSWORD}" "${input_url}")"

if [[ "${existing_code}" == "404" ]]; then
  create_code="$(curl -ksS -o /dev/null -w '%{http_code}' \
    -u "admin:${SPLUNK_PASSWORD}" \
    https://127.0.0.1:8089/services/data/inputs/http \
    --data-urlencode "name=${INPUT_NAME}" \
    --data-urlencode "token=${PROOFLAYER_HEC_TOKEN}" \
    --data-urlencode "index=${INDEX_NAME}" \
    --data-urlencode "indexes=${INDEX_NAME}" \
    --data-urlencode "sourcetype=prooflayer:sysmon" \
    --data-urlencode "disabled=0" \
    --data-urlencode "useACK=0")"

  if [[ "${create_code}" != "201" && "${create_code}" != "200" ]]; then
    echo "HEC input creation returned HTTP ${create_code}." >&2
    exit 1
  fi
  echo "Created HEC input ${INPUT_NAME}."
elif [[ "${existing_code}" == "200" ]]; then
  echo "HEC input ${INPUT_NAME} already exists."
else
  echo "HEC input lookup returned HTTP ${existing_code}." >&2
  exit 1
fi

health_code="$(curl -ksS -o /dev/null -w '%{http_code}' \
  -H "Authorization: Splunk ${PROOFLAYER_HEC_TOKEN}" \
  https://127.0.0.1:8088/services/collector/health)"

if [[ "${health_code}" != "200" ]]; then
  echo "HEC health check returned HTTP ${health_code}." >&2
  exit 1
fi

echo "PASS: dedicated HEC input is enabled and authenticated."
