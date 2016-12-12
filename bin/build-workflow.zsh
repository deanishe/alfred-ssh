#!/usr/bin/env zsh

wffiles=(assh info.plist README.md LICENCE.txt)
# Icons
icons=(icon.png update.png)
wffiles+=($icons)

workdir="$( cd "$( dirname "$0" )"/../; pwd )"
builddir="${workdir}/build"
distdir="${workdir}/dist"
delbuild=1

log() {
    echo "$@" > /dev/stderr
}

do_dist=0
do_build=1
do_copy=1
do_clean=0

usage() {
    cat <<EOS
build-workflow.zsh [options]

Build the .alfredworkflow file

Usage:
    build-workflow.zsh [-c] [-d]
    build-workflow.zsh -x
    build-workflow.zsh -h

Options:
    -c      Only copy assets to root (don't build workflow)
    -d      Also build distributable .alfredworkflow file
    -x      Clean build and dist directories
    -h      Show this help message and exit
EOS
}

while getopts ":cdhx" opt; do
  case $opt in
    c)
      do_build=0
      ;;
    d)
      do_dist=1
      ;;
    h)
      usage
      exit 0
      ;;
    x)
      do_clean=1
      ;;
    \?)
      log "Invalid option: -$OPTARG"
      exit 1
      ;;
  esac
done
shift $((OPTIND-1))

cleanup() {
    log "Cleaning up ..."
    test -d "$distdir" && rm -rf "$distdir"
    test -d "$builddir" && rm -rf "$builddir"
}

pushd "$workdir" &>/dev/null


if [[ $do_clean -eq 1 ]]; then
    cleanup
    popd &>/dev/null
    exit 0
fi


if [[ $do_copy -eq 1 ]]; then
    log "Copying icons to root ..."

    for f in $icons; do
        n="${f:t:r}"
        cp -v "icons/${f}" "./${n}.png"
    done
fi

if [[ $do_build -eq 1 ]]; then

    log "Building executable(s) ..."
    go build -v -o ./assh ./cmd/assh
    ST_BUILD=$?
    if [ "$ST_BUILD" != 0 ]; then
        log "Error building executable."
        cleanup
        popd &> /dev/null
        exit $ST_BUILD
    fi

    chmod 755 ./assh

    log

    log "Cleaning $builddir ..."
    rm -rvf "$builddir"

    log

    mkdir -vp "$builddir"

    for n in $wffiles; do
        cp -v "$n" "$builddir"
    done

    log
fi

if [[ $do_dist -eq 1 ]]; then

    log "Cleaning $distdir ..."
    rm -rvf "$distdir"

    # Get the dist filename from the executable
    zipfile="$( ./assh print distname 2>/dev/null )"
    if [ "$?" -ne 0 ]; then
        log "Error getting distname from alfssh."
        cleanup
        exit 1
    fi

    log

    # if test -e "$zipfile"; then
    #     log "Removing existing .alfredworkflow file ..."
    #     rm -rvf "$zipfile"
    #     log
    # fi

    pushd "$builddir" &>/dev/null

    log "Cleaning info.plist ..."
    /usr/libexec/PlistBuddy -c 'Delete :variables:DEMO_MODE' info.plist

    log "Building .alfredworkflow file ..."
    mkdir -vp "$distdir"
    zip "${distdir}/${zipfile}" ./*
    ST_ZIP=$?
    if [ "$ST_ZIP" != 0 ]; then
        log "Error creating .alfredworkflow file."
        popd &>/dev/null
        cleanup
        popd &>/dev/null
        exit $ST_ZIP
    fi
    popd &>/dev/null

    log
    log "Wrote '${zipfile}' in '${distdir}'"
fi

popd &>/dev/null
