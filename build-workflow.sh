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
ST_BUILD=$?
if [ "$ST_BUILD" != 0 ]; then
    log "Error building executable."
    rm -rf ./build/
    popd
    exit $ST_BUILD
fi

chmod 755 ./alfssh
cp -v ./alfssh ./build/alfssh

log "Building .alfredworkflow file ..."
pushd ./build/
zip -v ../Alfred-SSH.alfredworkflow *
ST_ZIP=$?
if [ "$ST_ZIP" != 0]; then
    log "Error creating .alfredworkflow file."
    rm -rf ./build/
    popd
    exit $ST_ZIP
fi
popd

log "Cleaning up ..."
rm -rvf ./build/

popd
log "All done."

