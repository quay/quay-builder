#! /bin/sh
set -e

# Write out certificate if one was given
if [[ -n "${CA_CERT}" ]]; then
    echo "[INFO]: CA_CERT found, writing out to /certs/cacert.crt"
    echo "${CA_CERT}" > /certs/cacert.crt
fi

# Start podman service
PODMAN_OPTS="--log-level=error"
if [[ "$DEBUG" == "true" ]]; then
    PODMAN_OPTS="--log-level=debug"
fi
podman $PODMAN_OPTS system service --time 0 &

# Ensure socket exists 
sleep 5s
RETRIES=5
while [[ ! -S '/tmp/podman-run-1000/podman/podman.sock' ]]
do
    if [[ $RETRIES -eq 0 ]]; then
        echo "[ERROR]: podman socket not found, exiting"
        exit 1
    fi
    echo "[INFO]: Waiting for podman to start. Checking again in 10s..."
    sleep 10s
    RETRIES=$((RETRIES - 1))
done

exec quay-builder
