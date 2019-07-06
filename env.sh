#!/bin/bash

# Source this file to export expected Alfred variables to environment

# getvar <name> | Read a value from info.plist
getvar() {
    local v="$1"
    /usr/libexec/PlistBuddy -c "Print :$v" info.plist
}

export alfred_workflow_bundleid=$( getvar "bundleid" )
export alfred_workflow_version=$( getvar "version" )
export alfred_workflow_name=$( getvar "name" )
export alfred_debug='1'

export alfred_workflow_cache="${HOME}/Library/Caches/com.runningwithcrayons.Alfred/Workflow Data/${alfred_workflow_bundleid}"
export alfred_workflow_data="${HOME}/Library/Application Support/Alfred/Workflow Data/${alfred_workflow_bundleid}"

# Alfred 3 environment if Alfred 4+ prefs file doesn't exist.
if [[ ! -f "$HOME/Library/Application Support/Alfred/prefs.json" ]]; then
    export alfred_workflow_cache="${HOME}/Library/Caches/com.runningwithcrayons.Alfred-3/Workflow Data/${alfred_workflow_bundleid}"
    export alfred_workflow_data="${HOME}/Library/Application Support/Alfred 3/Workflow Data/${alfred_workflow_bundleid}"
    export alfred_version="3.8.1"
fi


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

