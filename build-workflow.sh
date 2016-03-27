#!/bin/bash

here="$( cd "$( dirname "$0" )"; pwd )"


log() {
    echo "$@" > /dev/stderr
}

pushd "$here" &> /dev/null

log "Cleaning ./build ..."
rm -rvf ./build

log

log "Copying assets to ./build ..."

mkdir -vp ./build

cp -v icon.png ./build/
cp -v info.plist ./build/
cp -v README.md ./build/
cp -v LICENCE.txt ./build/

log

log "Building executable(s) ..."
go build -v -o ./alfssh ./alfssh.go
ST_BUILD=$?
if [ "$ST_BUILD" != 0 ]; then
    log "Error building executable."
    rm -rf ./build/
    popd &> /dev/null
    exit $ST_BUILD
fi

chmod 755 ./alfssh
cp -v ./alfssh ./build/alfssh

# Get the dist filename from the executable
zipfile="$(./alfssh --distname 2> /dev/null)"

log

if test -e "$zipfile"; then
    log "Removing existing .alfredworkflow file ..."
    rm -rvf "$zipfile"
    log
fi

log "Building .alfredworkflow file ..."
pushd ./build/ &> /dev/null
zip -v "../${zipfile}" *
ST_ZIP=$?
if [ "$ST_ZIP" != 0 ]; then
    log "Error creating .alfredworkflow file."
    rm -rf ./build/
    popd &> /dev/null
    exit $ST_ZIP
fi
popd &> /dev/null

log

log "Cleaning up ..."
rm -rvf ./build/

popd &> /dev/null
log "All done."

