#!/bin/bash -e

here="$( cd "$( dirname "$0" )"; pwd )"


log() {
    echo "$@" > /dev/stderr
}

pushd "$here" &> /dev/null

# Get metadata from info.plist
# bundleid="$( /usr/libexec/PlistBuddy -c 'Print :bundleid' info.plist )"
#
# if test -z "$bundleid"; then
#     log "No bundle ID found in info.plist"
#     exit 1
# fi
#
# name="$( /usr/libexec/PlistBuddy -c 'Print :name' info.plist )"
#
# if test -z "$name"; then
#     log "No workflow name found in info.plist"
#     exit 1
# fi


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
go build -v -o ./alfssh
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
if [ "$?" -ne 0 ]; then
    log "Error getting distname from alfssh."
    exit 1
fi

log

if test -e "$zipfile"; then
    log "Removing existing .alfredworkflow file ..."
    rm -rvf "$zipfile"
    log
fi

pushd ./build/ &> /dev/null

log "Cleaning info.plist ..."
/usr/libexec/PlistBuddy -c 'Delete :variables:DEMO_MODE' info.plist

log "Building .alfredworkflow file ..."
zip "../${zipfile}" ./*
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

log "Wrote '${zipfile}' in '$( pwd )'"

popd &> /dev/null
