#! /bin/sh
set -e

# This directory is for any custom certificates users want to mount
echo "Copying custom certs to trust if they exist"
if [ "$(ls -A /certs)" ]; then
    cp /certs/* /etc/pki/ca-trust/source/anchors
fi

update-ca-trust extract

# Update the defaule bundle to link to the newly generated bundle (not sure why /etc/pki/ca-trust/extracted/pem is not being updated...)
cat /certs/ssl.cert >> /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem
