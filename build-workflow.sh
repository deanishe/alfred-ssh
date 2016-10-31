#!/usr/bin/env zsh

wffiles=(alfssh icon.png info.plist README.md LICENCE.txt)

here="$( cd "$( dirname "$0" )"; pwd )"

log() {
    echo "$@" > /dev/stderr
}

cleanup() {
    local p="${here}/build"
    log "Cleaning up ..."
    test -d "$p" && rm -rf ./build/
}

pushd "$here" &> /dev/null

log "Building executable(s) ..."
go build -v -o ./alfssh ./cmd/alfssh
ST_BUILD=$?
if [ "$ST_BUILD" != 0 ]; then
    log "Error building executable."
    cleanup
    popd &> /dev/null
    exit $ST_BUILD
fi

chmod 755 ./alfssh
# cp -v ./alfssh ./build/alfssh

log

log "Cleaning ./build ..."
rm -rvf ./build

log

log "Copying assets to ./build ..."

mkdir -vp ./build

for n in $wffiles; do
    cp -v "$n" ./build/
done

log

# Get the dist filename from the executable
zipfile="$(./alfssh print distname 2> /dev/null)"
if [ "$?" -ne 0 ]; then
    log "Error getting distname from alfssh."
    cleanup
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
    popd &> /dev/null
    cleanup
    popd &> /dev/null
    exit $ST_ZIP
fi
popd &> /dev/null

log

cleanup

log "Wrote '${zipfile}' in '$( pwd )'"

popd &> /dev/null
