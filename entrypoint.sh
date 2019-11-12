#! /bin/sh
set -e

sh /load_extra_ca.sh
exec quay-builder
