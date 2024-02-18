
hostname_config_menu() {
    [ -f /etc/hostname ] && HOSTNAME=$(cat /etc/hostname)

    HOSTNAME=$(inputbox "Enter this system's host name" "$HOSTNAME")
    # If we're in the installer, just write the hostname file
    if [[ -n "$LUNAR_INSTALL" && -n "$HOSTNAME" ]]
    then
        echo "$HOSTNAME" > /etc/hostname
    else
        hostnamectl set-hostname "$HOSTNAME"
    fi
}

dev_add_menu() {
    local devices=()
    local devices_menu=()
    local dev

    if devices=($(get_unconfigured_dev_list))
    then
        : configure network devices
    else
        msgbox "Network devices" "No network devices found or all network devices have already been configured."
        return
    fi

    local i=1
    for dev in ${devices[@]}
    do
        devices_menu+=($i $dev)
        ((i++))
    done

    local PROMPT
    PROMPT="Select an interface to configure"
    result=$($DIALOG --title "Setup network interface" \
                     --ok-label "Select" \
                     --cancel-label "Back" \
                     --menu \
                     $PROMPT \
                     0 0 0 \
                     "${devices_menu[@]}") || return
    dev=${devices[$[result-1]]}

    dev_config_menu $dev
}

dev_config_menu() {
    local device=$1

    local choice

    local config_file="$CONFIG_DIR/${device}.network"

    eval "$(get_dev_config $device)"

    while true
    do
        choice=$($DIALOG --title "Network configuration: $device" \
                         --ok-label "Select" \
                         --cancel-label "Back" \
                         --menu "" 0 0 0 \
                         D "DHCP enabled?    [$($DHCP_enabled && echo Y || echo N)]" \
                         $(
                            if ! $DHCP_enabled
                            then
                                echo "I"
                                echo "IP Address     [$IP_Address]"
                                echo "N"
                                echo "Netmask        [$Netmask]"
                                echo "G"
                                echo "Gateway        [$Gateway]"
                                echo "S"
                                echo "Nameserver 1   [$DNS1]"
                                echo "T"
                                echo "Nameserver 2   [$DNS2]"
                            fi
                         )
                ) || return
        case "$choice" in
            D)
                if $DHCP_enabled
                then
                    DHCP_enabled=false
                else
                    DHCP_enabled=true
                fi
            ;;
            I) IP_Address=$(inputbox "Enter IP address" "$IP_Address") ;;
            N) Netmask=$(inputbox "Enter net mask" "$Netmask")         ;;
            G) Gateway=$(inputbox "Enter gateway" "$Gateway")          ;;
            S) DNS1=$(inputbox "Enter DNS server #1" "$DNS1")          ;;
            T) DNS2=$(inputbox "Enter DNS server #2" "$DNS2")          ;;
        esac

        if $DHCP_enabled
        then
            set_dev_config $device dhcp
        else
            set_dev_config $device static $IP_Address $Netmask $Gateway "$DNS1" "$DNS2"
        fi
    done
}

main_menu() {
    while true
    do
        COUNTER=0
        unset LIST
        unset MANAGE
        for DEVICE in $(get_configured_dev_list)
        do
            if [ -L $CONFIG_DIR/$DEVICE ]
            then
                continue
            fi
            STATUS=$(get_dev_status $DEVICE)
            INTERFACES[$COUNTER]=$DEVICE
            LIST+="$COUNTER\nEdit device $DEVICE    $STATUS\n"
            ((COUNTER++))
        done

        if (( COUNTER > 0 ))
        then
            MANAGE="M\nManage network devices\n"
        fi
		COMMAND=$($DIALOG  --title "Network configuration"  \
						  --ok-label "Select"               \
						  --cancel-label "Exit"             \
						  --menu                            \
						  ""                                \
						  0 0 0                             \
						  $(echo -en $LIST)                 \
						  "A"  "Add a network device"       \
						  "N"  "Setup host name"            \
						  $(echo -en $MANAGE)) || return
		case "$COMMAND" in
            [0-9]*) dev_config_menu ${INTERFACES[$COMMAND]}    ;;
            A)      dev_add_menu                               ;;
            D)      dns_config_menu                            ;;
            N)      hostname_config_menu                       ;;
            M)      ethernet_manage_menu                       ;;
		esac
    done
}

