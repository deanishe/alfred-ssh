#!/bin/bash

here="$( cd "$( dirname "$0" )"; pwd )"


log() {
    echo "$@" > /dev/stderr
}

pushd "$here"

log "Cleaning ./build ..."
rm -rvf ./build

log "Copying assets to ./build ..."

mkdir -vp ./build

cp -v icon.png ./build/
cp -v info.plist ./build/
cp -v README.md ./build/
cp -v LICENCE.txt ./build/

log "Building executable(s) ..."
go build -v -o ./alfssh ./alfssh.go
cp -v ./alfssh ./build/alfssh

log "Building .alfredworkflow file ..."
pushd ./build/
zip -v ../Alfred-SSH.alfredworkflow *
popd

log "Cleaning up ..."
rm -rvf ./build/

popd
log "All done."

