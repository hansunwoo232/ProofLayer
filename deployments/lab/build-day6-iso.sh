#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
WORKSPACE_ROOT="$(cd "${PROJECT_ROOT}/../.." && pwd)"
STATE_DIR="${PROOFLAYER_LAB_STATE_DIR:-${WORKSPACE_ROOT}/work/prooflayer-lab}"
SPLUNK_ENV="${PROJECT_ROOT}/deployments/splunk/.env"
SOURCE_DIR="${SCRIPT_DIR}/windows"
BASE_MEDIA="${STATE_DIR}/tools-iso"
STAGING_DIR="${STATE_DIR}/day6-tools-iso"
OUTPUT_ISO="${STATE_DIR}/prooflayer-day6.iso"
TEMP_ISO="${STATE_DIR}/prooflayer-day6.$$.iso"
VIRTIO_ZIP="${STATE_DIR}/virtio-win-prewhql-0.1.285.zip"
VIRTIO_ZIP_SHA256="db3b687b2863efc874e31520f3aed07fc106b1cad295dfa0a86f96855b0fc8ab"

if [[ ! -f "${SPLUNK_ENV}" ]]; then
  echo "Missing Splunk environment file: ${SPLUNK_ENV}" >&2
  exit 1
fi

set -a
source "${SPLUNK_ENV}"
set +a

if [[ -z "${PROOFLAYER_HEC_TOKEN:-}" || ${#PROOFLAYER_HEC_TOKEN} -lt 32 ]]; then
  echo "Missing valid PROOFLAYER_HEC_TOKEN. Run deployments/splunk/configure-hec.sh first." >&2
  exit 1
fi

for required in Sysmon64a.exe Eula.txt; do
  if [[ ! -f "${BASE_MEDIA}/${required}" ]]; then
    echo "Missing verified base media file: ${BASE_MEDIA}/${required}" >&2
    exit 1
  fi
done

if [[ ! -f "${VIRTIO_ZIP}" ]]; then
  echo "Missing verified VirtIO-Win package: ${VIRTIO_ZIP}" >&2
  exit 1
fi

actual_virtio_sha256="$(shasum -a 256 "${VIRTIO_ZIP}" | awk '{print $1}')"
if [[ "${actual_virtio_sha256}" != "${VIRTIO_ZIP_SHA256}" ]]; then
  echo "VirtIO-Win package SHA-256 mismatch." >&2
  echo "Expected: ${VIRTIO_ZIP_SHA256}" >&2
  echo "Actual:   ${actual_virtio_sha256}" >&2
  exit 1
fi

rm -rf "${STAGING_DIR}"
mkdir -p "${STAGING_DIR}"
cp "${BASE_MEDIA}/Sysmon64a.exe" "${BASE_MEDIA}/Eula.txt" "${STAGING_DIR}/"
cp "${SOURCE_DIR}/bootstrap.ps1" \
  "${SOURCE_DIR}/sysmon-lab.xml" \
  "${SOURCE_DIR}/install-netkvm.ps1" \
  "${SOURCE_DIR}/invoke-day6-proof.ps1" \
  "${SOURCE_DIR}/run-day6.ps1" \
  "${STAGING_DIR}/"

mkdir -p "${STAGING_DIR}/NetKVM/Win11/ARM64"
unzip -j "${VIRTIO_ZIP}" \
  'Win11/ARM64/netkvm.cat' \
  'Win11/ARM64/netkvm.inf' \
  'Win11/ARM64/netkvm.sys' \
  'Win11/ARM64/netkvmco.exe' \
  'Win11/ARM64/netkvmp.exe' \
  -d "${STAGING_DIR}/NetKVM/Win11/ARM64" >/dev/null

umask 077
printf "\$env:PROOFLAYER_HEC_TOKEN = '%s'\n" "${PROOFLAYER_HEC_TOKEN}" \
  > "${STAGING_DIR}/prooflayer-lab-secret.ps1"

rm -f "${TEMP_ISO}"
trap 'rm -f "${TEMP_ISO}"' EXIT
hdiutil makehybrid -iso -joliet -o "${TEMP_ISO}" "${STAGING_DIR}" >/dev/null
mv "${TEMP_ISO}" "${OUTPUT_ISO}"
trap - EXIT

echo "Created local Day 6 media: ${OUTPUT_ISO}"
echo "SHA-256: $(shasum -a 256 "${OUTPUT_ISO}" | awk '{print $1}')"
