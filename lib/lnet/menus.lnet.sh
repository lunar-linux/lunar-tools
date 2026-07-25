
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

wifi_strength_string() {
    local wifi_strength=$1

    local wifi_strength_str
    declare -a wifi_strength_str
    wifi_strength_str=(
        "[OOOOOOO]"
        "[OOOOOO.]"
        "[OOOOO..]"
        "[OOOO...]"
        "[OOO....]"
        "[OO.....]"
        "[O......]"
    )

    if ((wifi_strength > -55))
    then
        echo "${wifi_strength_str[0]}"
        return
    fi

    if ((wifi_strength > -60))
    then
        echo "${wifi_strength_str[1]}"
        return
    fi

    if ((wifi_strength > -65))
    then
        echo "${wifi_strength_str[2]}"
        return
    fi

    if ((wifi_strength > -70))
    then
        echo "${wifi_strength_str[3]}"
        return
    fi

    if ((wifi_strength > -75))
    then
        echo "${wifi_strength_str[4]}"
        return
    fi

    if ((wifi_strength > -80))
    then
        echo "${wifi_strength_str[5]}"
        return
    fi

    echo "${wifi_strength_str[6]}"
}

wifi_scan_menu() {
    local device=$1
    local index=0
    local menu
    local aps
    local ap_falgs
    declare -a menu
    declare -a aps
    declare -a ap_flags

    local tempfile=$(mktemp lnet_wifiXXXX)

    menu=()
    aps=()
    ap_flags=()

    $DIALOG --infobox "Scanning for wi-fi APs..." 0 0

    wifi_sort_aps $device > $tempfile

    while read line
    do
        eval line_items=($(echo $line) )
        ssid=${line_items[0]}
        mac=${line_items[1]}
        flags=${line_items[2]}
        level=${line_items[3]}

        if [ -z "$ssid" ]
        then
            ssid=" "
        fi

        level_str=$(wifi_strength_string $level)
        menu+=($index "$(printf '%-24s %s\n' $ssid $level_str)")
        aps+=($(printf '%s %s' $ssid $mac))
        ap_flags+=($flags)
        ((index++))
    done < $tempfile

    rm $tempfile

    local PROMPT="Select wifi AP"
    result=$($DIALOG --title "Select wifi AP" \
                     --ok-label "Select" \
                     --cancel-label "Back" \
                     --menu \
                     $PROMPT \
                     0 0 0 \
                     "${menu[@]}") || return 1

    echo ${aps[$result]} ${ap_flags[$result]}
}

wifi_get_password() {
    local ap=$1
    local PASSWORD

    PASSWORD=$($DIALOG --insecure \
                       --cancel-label "Show password" \
                        --passwordbox "Enter password for AP '$ap'" 0 0) ||
    PASSWORD=$($DIALOG --inputbox "Enter password for AP '$ap'" 0 0)

    echo $PASSWORD
}

wifi_select_menu() {
    local device=$1
    eval local ap_details=($(wifi_scan_menu $device)) || return 1
    local ap_flags
    local ap_password

    case "${ap[0]}" in
        [0-9a-f][0-9a-f]:[0-9a-f][0-9a-f]:[0-9a-f][0-9a-f]:*) # anonymous AP
            ap_name="${ap_details[0]}"
            ap_flags="${ap_details[1]}"
        ;;

        *)
            ap_name="${ap_details[0]}"
            ap_flags="${ap_details[2]}"
        ;;
    esac

    wifi_create_config $device

    if [[ $ap_flags =~ WEP || $ap_flags =~ WPA ]]
    then
        ap_password=$(wifi_get_password $ap_name)
        wifi_ap_password "$device" "$ap_name" "$ap_password"
    fi

    wifi_activate "$device"
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

    set_dev_config $dev dhcp

    dev_config_menu $dev
}

dev_edit_menu() {
    local device=$1

    PROMPT="Actions for interface $device\nDevice is: $(get_dev_status $device)"
    COMMAND=`$DIALOG  --title "Modify device $device" \
                      --ok-label "Select"             \
                      --cancel-label "Exit"           \
                      --menu                          \
                      $PROMPT                         \
                      0 0 0                           \
                      'C'  'Reconfigure'              \
                      'M'  'Manage'                   \
                      'D'  'Delete'`

    if [ $? != 0 ] ; then
      return
    fi

    case $COMMAND  in
        C) dev_config_menu $device ;;
        M) dev_manage_menu $device ;;
        D)
            if confirm "Are you sure you wish to delete $device?" "--defaultno"; then
                delete_dev_config $device
            fi
        ;;
    esac
}

# Manage all the devices
devices_manage_menu() {
    local DEVICE
    local STATUS
    local INTERFACES
    local LIST
    local COUNTER

    while true
    do
        LIST=()
        COUNTER=0
        for DEVICE in $(get_configured_dev_list)
        do
            STATUS=$(get_dev_status $DEVICE)
            INTERFACES[$COUNTER]=$DEVICE
            STR=$(printf "%-15s %-6s" $DEVICE $(get_dev_status $DEVICE))
            LIST+=("$COUNTER" "$STR")
            ((COUNTER++))
        done

        if (( COUNTER == 0 ))
        then
            msgbox "Manage Devices" \
                   "There are no interfaces to be listed. You may want to configure a device first." \
                   7
            return
        fi


        DEVICE="$($DIALOG --title "Manage devices" \
                         --ok-label "Select" \
                         --cancel-label "Return" \
                         --menu \
                         "Select a device to manage" \
                         0 0 0 \
                         "${LIST[@]}")" || return
        dev_manage_menu ${INTERFACES[$DEVICE]}
    done
}

# Manage a single device
dev_manage_menu() {
    local device=$1

    while true
    do
        STATUS=$(get_dev_status $1)
        if [[ "$STATUS" == "[ UP ]" ]]
        then
            TOGGLE="Stop"
        else
            TOGGLE="Start"
        fi

        COMMAND=$($DIALOG --title "Manage Device $device"     \
                          --cancel-label "Return"             \
                          --menu "Device $device is: $STATUS" \
                          0 0 0                               \
                          "S" "$TOGGLE Device"                \
                          "R" "Restart Device") || return
        case "$COMMAND" in
            S)
                case "$TOGGLE" in
                    Start)
                        dev_up $device
                    ;;

                    Stop)
                        dev_down $device
                    ;;
                esac

                ;;
            R)
                dev_down $device
                dev_up $device
            ;;
        esac
    done
}

# Manually specify IP address, netmask, gateway and DNS
config_device_manual() {
    local device=$1
    local changed=false

    # set IP_Address, Netmask, Gateway, DNS1, DNS2
    eval $(get_dev_config $device)

    local old_IP_Address old_Netmask old_Gateway old_DNS1 old_DNS2

    while true
    do
        choice=$($DIALOG --title "Manual network configuration: $device" \
                         --ok-label "Edit"                               \
                         --extra-button --extra-label "Back"             \
                         --cancel-label "Save"                           \
                         --menu "" 0 0 0                                 \
                         "I" "IP Address     [$IP_Address]"              \
                         "N" "Netmask        [$Netmask]"                 \
                         "G" "Gateway        [$Gateway]"                 \
                         "S" "Nameserver 1   [$DNS1]"                    \
                         "T" "Nameserver 2   [$DNS2]")
        case $? in
            1)  # "Save" selected
                break
            ;;

            3)  # "Back" selected--just go back to the config style
                # selector
                dev_config_menu $device
                return
            ;;
        esac

        case ${choice} in
            I)
                old_IP_Address=$IP_Address
                IP_Address=$(inputbox "Enter IP address" "$IP_Address")
                [[ $old_IP_Address != $IP_Address ]] && changed=true
                # if someone enters a CIDR in their enthusiasm, just
                # calculate the netmask from it
                case "$IP_Address" in
                    */*)
                        Netmask=$(cidr2mask ${IP_Address#*/})
                        IP_Address=${IP_Address%/*}
                    ;;
                esac
            ;;

            N)
                old_Netmask=$Netmask
                Netmask=$(inputbox "Enter net mask" "$Netmask")
                [[ $old_Netmask != $Netmask ]] && changed=true
            ;;

            G)
                old_Gateway=$Gateway
                Gateway=$(inputbox "Enter gateway" "$Gateway")
                [[ $old_Gateway != $Gateway ]] && changed=true
            ;;

            S)
                old_DNS1=$DNS1
                DNS1=$(inputbox "Enter DNS server #1" "$DNS1")
                [[ $old_DNS1 != $DNS1 ]] && changed=true
            ;;

            T)
                old_DNS2=$DNS2
                DNS2=$(inputbox "Enter DNS server #2" "$DNS2")
                [[ $old_DNS2 != $DNS2 ]] && changed=true
            ;;
        esac
    done

    if $changed
    then
        $DIALOG --title "Manual network configuration: $device" \
                --yes-label Confirm \
                --no-label  Cancel \
                --no-collapse \
                --yesno "
Confirm network configuration:

IP Address:    $IP_Address
Netmask:       $Netmask
Gateway:       $Gateway
Nameserver 1:  $DNS1
Nameserver 2:  $DNS2" \
               0 0

        if [ $? -eq 0 ]
        then
            set_dev_config $device static $IP_Address $Netmask $Gateway "$DNS1" "$DNS2"
        else
            return 1
        fi
    fi
}

dev_config_menu() {
    local device=$1
    local config_file="$CONFIG_DIR/${device}.network"
    local choice
    local dialog_answer
    local dhcp_on
    local manual_on

    # set Wifi_Device et al
    eval "$(get_dev_config $device)"

    if ${DHCP_enabled:-true}
    then
        dhcp_on=ON
        manual_on=OFF
    else
        dhcp_on=OFF
        manual_on=ON
    fi

    # Before starting to worry about IP addresses and DHCP for a WiFi device,
    # make sure you connect to an AP first.
    if $WiFi_Device
    then
        wifi_select_menu $device || return 1
    fi

    choice=$($DIALOG --title "Network configuration: $device"              \
             --radiolist "Choose a network configuration style"            \
                 0 0 0                                                     \
                 DHCP 'Configure network automatically' ${dhcp_on}         \
                 Manual                                                    \
                    'Specify IP address, subnet, gateway and DNS settings' \
                ${manual_on})

    case "${choice}" in
        DHCP)
            set_dev_config $device dhcp
        ;;

        Manual)
            config_device_manual $device || return 1
        ;;

        *) # if user hits cancel, don't configure device after all
            return 1
        ;;
    esac
    return 0
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
                          $(echo -en $MANAGE)) || return 0
        case "$COMMAND" in
            [0-9]*) dev_edit_menu ${INTERFACES[$COMMAND]}      ;;
            A)      dev_add_menu                               ;;
            D)      dns_config_menu                            ;;
            N)      hostname_config_menu                       ;;
            M)      devices_manage_menu                       ;;
        esac
    done
}
