#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"
ROLE_NAME="prooflayer_observer"
USER_NAME="prooflayer_observer"
INDEX_NAME="prooflayer_test"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Missing ${ENV_FILE}. Run ./setup.sh first." >&2
  exit 1
fi

set -a
source "${ENV_FILE}"
set +a

if [[ -z "${PROOFLAYER_OBSERVER_PASSWORD:-}" || ${#PROOFLAYER_OBSERVER_PASSWORD} -lt 32 ]]; then
  PROOFLAYER_OBSERVER_PASSWORD="$(openssl rand -hex 24)"
  temporary_env="$(mktemp "${SCRIPT_DIR}/.env.tmp.XXXXXX")"
  chmod 600 "${temporary_env}"
  grep -v '^PROOFLAYER_OBSERVER_PASSWORD=' "${ENV_FILE}" > "${temporary_env}" || true
  printf 'PROOFLAYER_OBSERVER_PASSWORD=%s\n' "${PROOFLAYER_OBSERVER_PASSWORD}" >> "${temporary_env}"
  mv "${temporary_env}" "${ENV_FILE}"
  echo "Generated and stored an ignored observer password."
fi

role_url="https://127.0.0.1:8089/services/authorization/roles/${ROLE_NAME}"
role_code="$(curl -ksS -o /dev/null -w '%{http_code}' \
  -u "admin:${SPLUNK_PASSWORD}" "${role_url}")"

if [[ "${role_code}" == "404" ]]; then
  create_role_code="$(curl -ksS -o /dev/null -w '%{http_code}' \
    -u "admin:${SPLUNK_PASSWORD}" \
    https://127.0.0.1:8089/services/authorization/roles \
    --data-urlencode "name=${ROLE_NAME}" \
    --data-urlencode "capabilities=search" \
    --data-urlencode "srchIndexesAllowed=${INDEX_NAME}" \
    --data-urlencode "srchIndexesDefault=${INDEX_NAME}")"
  if [[ "${create_role_code}" != "201" && "${create_role_code}" != "200" ]]; then
    echo "Observer role creation returned HTTP ${create_role_code}." >&2
    exit 1
  fi
  echo "Created role ${ROLE_NAME}."
elif [[ "${role_code}" == "200" ]]; then
  echo "Role ${ROLE_NAME} already exists."
else
  echo "Observer role lookup returned HTTP ${role_code}." >&2
  exit 1
fi

user_url="https://127.0.0.1:8089/services/authentication/users/${USER_NAME}"
user_code="$(curl -ksS -o /dev/null -w '%{http_code}' \
  -u "admin:${SPLUNK_PASSWORD}" "${user_url}")"

if [[ "${user_code}" == "404" ]]; then
  create_user_code="$(curl -ksS -o /dev/null -w '%{http_code}' \
    -u "admin:${SPLUNK_PASSWORD}" \
    https://127.0.0.1:8089/services/authentication/users \
    --data-urlencode "name=${USER_NAME}" \
    --data-urlencode "password=${PROOFLAYER_OBSERVER_PASSWORD}" \
    --data-urlencode "roles=${ROLE_NAME}")"
  if [[ "${create_user_code}" != "201" && "${create_user_code}" != "200" ]]; then
    echo "Observer user creation returned HTTP ${create_user_code}." >&2
    exit 1
  fi
  echo "Created user ${USER_NAME}."
elif [[ "${user_code}" == "200" ]]; then
  echo "User ${USER_NAME} already exists."
else
  echo "Observer user lookup returned HTTP ${user_code}." >&2
  exit 1
fi

auth_code="$(curl -ksS -o /dev/null -w '%{http_code}' \
  -u "${USER_NAME}:${PROOFLAYER_OBSERVER_PASSWORD}" \
  https://127.0.0.1:8089/services/server/info)"

if [[ "${auth_code}" != "200" ]]; then
  echo "Observer authentication returned HTTP ${auth_code}." >&2
  exit 1
fi

allowed_result="$(curl -ksS \
  -u "${USER_NAME}:${PROOFLAYER_OBSERVER_PASSWORD}" \
  https://127.0.0.1:8089/services/search/jobs/export \
  --data-urlencode "search=search index=${INDEX_NAME} | stats count" \
  --data "output_mode=json")"

if ! grep -Eq '"count"[[:space:]]*:[[:space:]]*"[0-9]+"' <<< "${allowed_result}"; then
  echo "Observer could not search ${INDEX_NAME}." >&2
  exit 1
fi

denied_result="$(curl -ksS \
  -u "${USER_NAME}:${PROOFLAYER_OBSERVER_PASSWORD}" \
  https://127.0.0.1:8089/services/search/jobs/export \
  --data-urlencode 'search=search index=_internal | stats count' \
  --data "output_mode=json")"

if grep -Eq '"count"[[:space:]]*:[[:space:]]*"[1-9][0-9]*"' <<< "${denied_result}"; then
  echo "Observer unexpectedly returned events from _internal." >&2
  exit 1
fi

echo "PASS: observer can search ${INDEX_NAME} and cannot return _internal events."
