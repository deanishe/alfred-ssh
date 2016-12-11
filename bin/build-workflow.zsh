#!/usr/bin/env zsh

wffiles=(alfssh info.plist README.md LICENCE.txt)
# Icons
icons=(icon.png update.png)
wffiles+=($icons)

workdir="$( cd "$( dirname "$0" )"/../; pwd )"
delbuild=1

log() {
    echo "$@" > /dev/stderr
}

do_dist=0
do_clean=0

usage() {
    cat <<EOS
build-workflow.zsh [options]

Build the .alfredworkflow file

Usage:
    build-workflow.zsh [-d] [-c]
    build-workflow.zsh -h

Options:
    -c      Clean the dist directory
    -d      Also build distributable .alfredworkflow file
    -h      Show this help message and exit
EOS
}

while getopts ":cdh" opt; do
  case $opt in
    c)
      do_clean=1
      ;;
    d)
      do_dist=1
      ;;
    h)
      usage
      exit 0
      ;;
    \?)
      log "Invalid option: -$OPTARG"
      exit 1
      ;;
  esac
done
shift $((OPTIND-1))

cleanup() {
    local p="${workdir}/dist"
    log "Cleaning up ..."
    test -d "$p" && rm -rf ./dist/
}

pushd "$workdir" &>/dev/null

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

log "Cleaning ./dist ..."
rm -rvf ./dist

log

log "Copying icons to root ..."

for f in $icons; do
    n="${f:t:r}"
    cp -v "icons/${f}" "./${n}.png"
done

mkdir -vp ./dist

for n in $wffiles; do
    cp -v "$n" ./dist/
done


log

if [[ $do_dist -eq 1 ]]; then
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

    pushd ./dist/ &>/dev/null

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
    popd &>/dev/null

    log
    log "Wrote '${zipfile}' in '$( pwd )'"
fi

[[ $do_clean -eq 1 ]] && cleanup


popd &>/dev/null
