#!/bin/bash

# Source this file to export expected Alfred variables to environment
# Needed to run modd or ./bin/build-workflow.zsh.

# getvar <name> | Read a value from info.plist
getvar() {
    local v="$1"
    /usr/libexec/PlistBuddy -c "Print :$v" info.plist
}

export alfred_workflow_bundleid=$( getvar "bundleid" )
export alfred_workflow_version=$( getvar "version" )
export alfred_workflow_name=$( getvar "name" )
export alfred_workflow_data="$HOME/Library/Application Support/Alfred 3/Workflow Data/$alfred_workflow_bundleid"
export alfred_workflow_cache="$HOME/Library/Caches/com.runningwithcrayons.Alfred-3/Workflow Data/$alfred_workflow_bundleid"
export alfred_debug='1'

export DISABLE_CONFIG=$( getvar "variables:DISABLE_CONFIG" )
export DISABLE_ETC_CONFIG=$( getvar "variables:DISABLE_ETC_CONFIG" )
export DISABLE_ETC_HOSTS=$( getvar "variables:DISABLE_ETC_HOSTS" )
export DISABLE_HISTORY=$( getvar "variables:DISABLE_HISTORY" )
export DISABLE_KNOWN_HOSTS=$( getvar "variables:DISABLE_KNOWN_HOSTS" )
export EXIT_ON_SUCCESS=$( getvar "variables:EXIT_ON_SUCCESS" )
export MOSH_CMD=$( getvar "variables:MOSH_CMD" )
export SFTP_APP=$( getvar "variables:SFTP_APP" )
export SSH_APP=$( getvar "variables:SSH_APP" )
export SSH_CMD=$( getvar "variables:SSH_CMD" )

