#!/bin/bash
set -e

# define architecture we want to build
ARCH=${ARCH:-"amd64 arm64"}
OS=${OS:-linux darwin}
TAG=${TAG:-0.0.0}

# clean up
echo "Running clean up..."
rm -rf output
rm -rf artifacts

if test -n "${DEST}"; then
    dest=`realpath ${DEST}`
fi

# build
# we want to build statically linked binaries
export CGO_ENABLED=0
echo -n "Building... "

for os in ${OS}; do
    for arch in ${ARCH}; do
        echo -n "${os}_${arch} "
        env GOOS=${os} GOARCH=${arch} go build -o "output/terraform-provider-jira_${TAG}_${os}_${arch}/terraform-provider-jira_${TAG}"
        cp README.md output/terraform-provider-jira_${TAG}_${os}_${arch}
        cp LICENSE output/terraform-provider-jira_${TAG}_${os}_${arch}


        if test -n "${dest}"; then
            mkdir -p "${dest}/.terraform.d/plugins/local/fceller/jira/${TAG}/${os}_${arch}"
            cp \
                "output/terraform-provider-jira_${TAG}_${os}_${arch}/terraform-provider-jira_${TAG}" \
                "${dest}/.terraform.d/plugins/local/fceller/jira/${TAG}/${os}_${arch}/terraform-provider-jira"
        fi
    done
done
echo

# Zip and copy to the dist dir
echo -n "Packaging... "
mkdir artifacts

for PLATFORM in $(find ./output -mindepth 1 -maxdepth 1 -type d); do
    OSARCH=$(basename ${PLATFORM})
    echo -n "${OSARCH} "

    pushd output/${OSARCH} >/dev/null 2>&1
    zip ../../artifacts/${OSARCH}.zip *
    popd >/dev/null 2>&1

    pushd artifacts >/dev/null 2>&1
    shasum -a 256 ${OSARCH}.zip >> terraform-provider-jira_${TAG}_SHA256SUMS
    popd >/dev/null 2>&1
done

pushd artifacts >/dev/null 2>&1
gpg --detach-sign terraform-provider-jira_${TAG}_SHA256SUMS 
popd >/dev/null 2>&1
