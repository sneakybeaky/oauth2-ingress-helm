#!/bin/sh

set -ex

IN_DIR=$1
OUT_DIR=$2

if [ ! -f "${VERSION_FILE}" ]; then
    echo "$0: File with version string not found : '${VERSION_FILE}' not found."
    exit 1
fi

helm repo add stable https://kubernetes-charts.storage.googleapis.com/
helm dependency build "${IN_DIR}"
helm package "${IN_DIR}" --version $(cat ${VERSION_FILE}) -d "${OUT_DIR}"
