# Configure network device
set_dev_config() {
    local device
    local config_type

    if [ -n "$1" ]
    then
        device=$1
    fi

    if [ -n "$2" ]
    then
        config_type=$2
    fi

    if [ $config_type == "dhcp" ]
    then
        {
            echo "[Match]"
            echo "Name=$device"
            echo
            echo "[Network]"
            echo "DHCP=yes"
        } > $CONFIG_DIR/${device}.network
        return
    fi

    if [ $config_type == "static" ]
    then
        local ipaddr=$3
        local gateway=$4
        local dns1=$5
        local dns2=$6

        {
            echo '[Match]'
            echo "Name=$device"
            echo
            echo '[Network]'
            echo "Address=$ipaddr"
            echo "Gateway=$gateway"
            if [ -n "$dns1" ]
            then
                echo "DNS=$dns1"
                if [ -n "$dns2" ]
                then
                    echo "DNS=$dns2"
                fi
            fi
        } > $CONFIG_DIR/${device}.network
    fi
}

# Get current network device configuration
#
# Outputs name=value pairs which can be used in an "eval" statement, like
#
#    eval $(get_dev_config enp3s0)
#
# If no /etc/systemd/network/${device}.network file is found,
# it outputs a set of defaults to set up the device with DHCP.
get_dev_config() {
    local device=$1

    local config_file="$CONFIG_DIR/${device}.network"
    local DHCP_enabled=true
    local IP_Address=0.0.0.0
    local Netmask=255.255.255.255
    local Gateway=0.0.0.0
    local CIDR
    local DNS1
    local DNS2
    local DNS
    local masklen=32

    if [ -f $config_file ]
    then
        if get_config_value Network DHCP $config_file >/dev/null
        then
            DHCP_enabled=true
        else
            DHCP_enabled=false
            if CIDR=$(get_config_value Network Address $config_file)
            then
                masklen=${CIDR#*/}
                IP_Address=${CIDR%/*}
                Netmask=$(cidr2mask $masklen)

            fi

            Gateway=$(get_config_value Network Gateway $config_file)

            DNS=( $(get_config_value Network DNS) )
            DNS1=${DNS[0]}
            DNS2=${DNS[1]}
        fi
    fi

    echo DHCP_enabled=$DHCP_enabled \
         IP_Address=$IP_Address \
         Netmask=$Netmask \
         Gateway=$Gateway \
         DNS1=$DNS1 \
         DNS2=$DNS2
}
