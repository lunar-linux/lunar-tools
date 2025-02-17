
# Get list of all network interfaces
get_dev_list() {
    ip link show | awk -F": " '/^[0-9]+/{ if($2 != "lo") print $2; }' | sort
}

# Get list of network interfaces with configuration files
get_configured_dev_list() {
    local interfaces=()
    local dev

    for dev in $(get_dev_list)
    do
        if [ -f $CONFIG_DIR/${dev}.network ]
        then
            interfaces+=($dev)
        fi
    done

    if ((${#interfaces} > 0))
    then
        for interface in ${interfaces[@]}
        do
            echo "$interface"
        done
        return 0
    else
        return 1
    fi
}

# Get list of network interfaces without configuration files
get_unconfigured_dev_list() {
    local interfaces=()
    local dev

    for dev in $(get_dev_list)
    do
        if [ ! -f $CONFIG_DIR/${dev}.network ]
        then
            interfaces+=($dev)
        fi
    done

    if ((${#interfaces} > 0))
    then
        for interface in ${interfaces[@]}
        do
            echo "$interface"
        done
        return 0
    else
        return 1
    fi
}

# Get whether device is up or down
get_dev_status() {
    local dev=$1

	if ip -j -p link show $dev | grep -q '^ *"flags":.*"UP"'
	then
        echo '[ UP ]'
	else
        echo '[DOWN]'
    fi
}

# Bring device up (if configured)
dev_up() {
    local dev=$1

    if [ -f $CONFIG_DIR/${dev}.network ]
    then
        networkctl up $dev
    fi
}

# Bring device down (if configured)
dev_down() {
    local dev=$1

    if [ -f $CONFIG_DIR/${dev}.network ]
    then
        networkctl down $dev
    fi
}
