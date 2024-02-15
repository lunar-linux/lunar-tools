
# Configure network device
netdev_config() {
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

