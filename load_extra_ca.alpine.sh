#! /bin/sh
set -e

# This directory is for any custom certificates users want to mount
echo "Copying custom certs to trust if they exist"
if [ "$(ls -A /certs)" ]; then
    cp /certs/* /usr/local/share/ca-certificates
fi

update-ca-certificates || true
