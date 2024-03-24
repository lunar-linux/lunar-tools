#!/bin/bash

# The kernel of a tiny TOML parser

# Get the entire contents of a TOML config file section
get_config_section() {
    local section=$1
    local file=$2

    sed -n -e '/^\['$section'\]/{:a;n;p;ba}' $file  | sed -ne '/^\[/q;p'
}

# Get a specific key from a specific section of TOML config file
#
# Returns an exit code of 1 if the key doesn't exist, outputs the
# value if it does exist
#
# If multiple key are found, output each one on a line by itself.
get_config_value() {
    local section=$1
    local key=$2
    local file=$3

    local value=$(get_config_section $section $file | grep "^$key=" | cut -f2 -d=)
    if [ -n "$value" ]
    then
        echo "$value"
    else
        return 1
    fi
}
