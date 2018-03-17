#!/usr/bin/env zsh

wffiles=(info.plist README.md LICENCE.txt)

workdir="$( cd "$( dirname "$0" )"/../; pwd )"
builddir="${workdir}/build"
distdir="${workdir}/dist"
delbuild=1

log() {
    echo "$@" > /dev/stderr
}

do_dist=false
do_build=true
do_clean=false

usage() {
    cat <<EOS
build-workflow.zsh [options]

Build the .alfredworkflow file

Usage:
    build-workflow.zsh [-c] [-d]
    build-workflow.zsh -x
    build-workflow.zsh -h

Options:
    -d      Also build distributable .alfredworkflow file
    -x      Clean build and dist directories
    -h      Show this help message and exit
EOS
}

while getopts ":dhx" opt; do
  case $opt in
    d)
      do_dist=true
      ;;
    h)
      usage
      exit 0
      ;;
    x)
      do_clean=true
      ;;
    \?)
      log "Invalid option: -$OPTARG"
      exit 1
      ;;
  esac
done
shift $((OPTIND-1))

cleanup() {
    log "cleaning up ..."
    test -d "$distdir" && rm -rf "$distdir"
    test -d "$builddir" && rm -rf "$builddir"
}

pushd "$workdir" &>/dev/null


$do_clean && {
    cleanup
    popd &>/dev/null
    exit 0
}

$do_build && {

    log "cleaning $builddir ..."
    rm -rvf "$builddir"/*

    log

    mkdir -vp "${builddir}/icons"

    log "building executable(s) ..."
    go build -v -o "${builddir}/assh" ./cmd/assh
    ST_BUILD=$?
    if [ "$ST_BUILD" != 0 ]; then
        log "error building executable."
        cleanup
        popd &> /dev/null
        exit $ST_BUILD
    fi

    chmod 755 "${builddir}/assh"

    log

    for n in $wffiles; do
        ln -v "$n" "$builddir"
    done

	command cp -vf "${workdir}/icons/"*.png "${builddir}/icons/"
	command mv -vf "${builddir}/icons/icon.png" "${builddir}/icon.png"
	command cp -vf "./icons/config.png" "${builddir}/A3CF9185-4D22-48D1-9515-851538E8D12B.png"

    log
}

$do_dist && {

    log "cleaning $distdir ..."
    rm -rvf "$distdir"

    # Get the dist filename from the executable
    zipfile="$( ./build/assh print distname 2>/dev/null )"
    if [ "$?" -ne 0 ]; then
        log "error getting distname from alfssh."
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

    log "cleaning info.plist ..."
    /usr/libexec/PlistBuddy -c 'Delete :variables:DEMO_MODE' info.plist

    log "building .alfredworkflow file ..."
    mkdir -vp "$distdir"
    zip -r "${distdir}/${zipfile}" ./*
    ST_ZIP=$?
    if [ "$ST_ZIP" != 0 ]; then
        log "error creating .alfredworkflow file."
        popd &>/dev/null
        cleanup
        popd &>/dev/null
        exit $ST_ZIP
    fi
    popd &>/dev/null

    log
    log "wrote '${zipfile}' in '${distdir}'"
}

popd &>/dev/null
