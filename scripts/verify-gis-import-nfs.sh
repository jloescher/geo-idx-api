#!/usr/bin/env bash
# Verify GIS import NFS/shared path visibility (run on re-db or re-node-02 as root).
set -euo pipefail

HOST_PATH="${GIS_IMPORT_HOST_PATH:-/data/coolify/gis-imports}"
CONTAINER_PATH="${GIS_IMPORT_PATH:-/var/cache/geoidx/gis-imports}"
EXPECTED_UID=65534

echo "== GIS import path check =="
echo "host path: ${HOST_PATH}"

if [[ ! -d "${HOST_PATH}" ]]; then
  echo "FAIL: ${HOST_PATH} does not exist"
  exit 1
fi

mount_info="$(mount | grep "${HOST_PATH}" || true)"
if [[ -n "${mount_info}" ]]; then
  echo "mount: ${mount_info}"
else
  echo "WARN: ${HOST_PATH} is not a separate mount (local disk or bind only)"
fi

owner="$(stat -c '%u:%g' "${HOST_PATH}")"
if [[ "${owner}" != "${EXPECTED_UID}:${EXPECTED_UID}" ]]; then
  echo "WARN: owner is ${owner}, expected ${EXPECTED_UID}:${EXPECTED_UID} (nobody)"
else
  echo "OK: owner ${owner}"
fi

probe="${HOST_PATH}/.verify-gis-import-$(date +%s)"
if echo "probe" >"${probe}" 2>/dev/null; then
  rm -f "${probe}"
  echo "OK: writable at ${HOST_PATH}"
else
  echo "FAIL: cannot write to ${HOST_PATH}"
  exit 1
fi

pinellas_zip="${HOST_PATH}/pinellas/Parcels.zip"
if [[ -f "${pinellas_zip}" ]]; then
  echo "OK: found $(ls -lh "${pinellas_zip}")"
else
  echo "INFO: no ${pinellas_zip} yet (upload not run or different source key/filename)"
fi

# Optional: check running idx-api-web / worker container mount
if command -v docker >/dev/null 2>&1; then
  while IFS= read -r cid; do
    [[ -z "${cid}" ]] && continue
    name="$(docker inspect -f '{{.Name}}' "${cid}" | tr -d '/')"
    dest="$(docker inspect -f '{{range .Mounts}}{{if eq .Destination "'"${CONTAINER_PATH}"'"}}{{.Source}} -> {{.Destination}} ({{.Type}}){{end}}{{end}}' "${cid}")"
    if [[ -n "${dest}" ]]; then
      echo "container ${name}: ${dest}"
    fi
  done < <(docker ps -q --filter "name=idx-api" 2>/dev/null || true)
fi

echo "Done."
