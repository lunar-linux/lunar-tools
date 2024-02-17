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
    local masklen=32

    if [ -f $config_file ]
    then
        if grep -A 2 '[Network]' $config_file | grep -q DHCP
        then
            DHCP_enabled=true
        else
            DHCP_enabled=false
            CIDR=$(grep -F -A 10 '[Network]' $config_file |
                      grep '^ *Address=' |
                      cut -f2 -d=)
            if [ -n "$CIDR" ]
            then
                masklen=${CIDR#*/}
                IP_Address=${CIDR%/*}
                Netmask=$(cidr2mask $masklen)
            fi

            Gateway=$(grep -F -A 10 '[Network]"' $config_file |
                      grep '^ *Gateway=' |
                      cut -f2 -d=)

            DNS1=$(grep -F -A 10 '[Network]"' $config_file |
                    grep '^ *DNS=' |
                    head -1 |
                    cut -f1 -d=)
            DNS2=$(grep -F -A 10 '[Network]"' $config_file |
                    grep '^ *DNS=' |
                    tail +2 |
                    head -1 |
                    cut -f1 -d=)

            if [ "$DNS2" == "$DNS1" ]
            then
                DNS2=""
            fi

        fi
    fi

    echo DHCP_enabled=$DHCP_enabled \
         IP_Address=$IP_Address \
         Netmask=$Netmask \
         Gateway=$Gateway \
         DNS1=$DNS1 \
         DNS2=$DNS2
}
