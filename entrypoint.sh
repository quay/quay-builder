#! /bin/sh
set -e

################################# Functions #################################

setup_kubernetes_podman(){
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
    RETRIES=5
    while [[ ! -S '/tmp/podman-run-1000/podman/podman.sock' ]]
    do
        if [[ $RETRIES -eq 0 ]]; then
            echo "[ERROR]: podman socket not found, exiting"
            exit 1
        fi
        echo "[INFO]: Waiting for podman to start. Checking again in 3s..."
        sleep 3s
        RETRIES=$((RETRIES - 1))
    done
}

load_extra_ca(){
    # This directory is for any custom certificates users want to mount
    echo "Copying custom certs to trust if they exist"
    if [ "$(ls -A /certs)" ]; then
        cp /certs/* /etc/pki/ca-trust/source/anchors
    fi

    update-ca-trust extract

    # Update the default bundle to link to the newly generated bundle (not sure why /etc/pki/ca-trust/extracted/pem is not being updated...)
    if [ -f "/certs/ssl.cert" ]; then
        cat /certs/ssl.cert >> /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem
    fi
}

################################# Begin execution #################################

case $EXECUTOR in 
    "kubernetesPodman")
        setup_kubernetes_podman
        ;;
    "popen" | "ec2" | "kubernetes" | "")
        load_extra_ca
        ;;
    *)
        echo "[ERROR]: Unrecognized executor: $EXECUTOR"
        exit 1
        ;;
esac

exec quay-builder
