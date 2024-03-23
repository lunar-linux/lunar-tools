#!/bin/bash

# wifi_find_aps <device>
#
# Use wpa_supplicant to ask the device to scan for wireless APs

wifi_find_aps() {
    local device=$1

    wpa_cli -i $device scan > /dev/null 2>&1
    sleep 5 # number chosen arbitrarily
    wpa_cli scan_result | awk '

        BEGIN {
            cell_num=0
            print "local WIFI_SSD"
            print "local WIFI_MAC"
            print "local WIFI_FLAGS"
            print "local WIFI_LEVEL"
        }

        /^[0-9a-f][0-9a-f]:[0-9a-f][0-9a-f]:[0-9a-f][0-9a-f]:/ {
            wifi_ssid = $5
            wifi_flags = $4
            wifi_level = $3
            wifi_freq = $2
            wifi_mac = $1
            printf "WIFI_SSID[%d]=\"%s\"\n", cell_num, wifi_ssid
            printf "WIFI_MAC[%d]=\"%s\"\n", cell_num, wifi_mac
            printf "WIFI_FLAGS[%d]=\"%s\"\n", cell_num, wifi_flags
            printf "WIFI_LEVEL[%d]=\"%s\"\n", cell_num, wifi_level

            cell_num++
        }

	END {
	    printf("local wifi_count=%d\n",cell_num-1)
	}'
}

# wifi_sort_aps
#
# Sort available APs in order of signal strength

wifi_sort_aps() {
    local device=$1

    eval $(wifi_find_aps $device)

    for i in $(seq 0 $wifi_count)
    do
        echo ${WIFI_LEVEL[$i]} ${i}
    done | sort -rn | awk '{ print $2 }' | while read index
    do
        echo ${WIFI_SSID[$index]:-\"\"} ${WIFI_MAC[$index]} ${WIFI_FLAGS[$index]} ${WIFI_LEVEL[$index]}
    done
}

wifi_create_config() {
    local device=$1
    local configfile=/etc/wpa_supplicant/wpa_supplicant-${device}.conf

    if [ ! -f $configfile ]
    then
        if [ ! -d $(dirname $configfile) ]
        then
            mkdir $(dirname $configfile)
        fi
    fi

    echo "ctrl_interface=/run/wpa_supplicant" > $configfile
    echo "ctrl_interface_group=0" >> $configfile
    echo "update_config=1" >> $configfile
    echo >> $configfile
}

# wifi_ap_password <device> <AP> <password>
#
# Creates a wpa_supplicant fragment
wifi_ap_password() {
    local device=$1
    local AP=$2
    local PASS=$3

    wpa_passphrase "$AP" "$PASS" >> /etc/wpa_supplicant/wpa_supplicant-${device}.conf
    chmod 600 /etc/wpa_supplicant/wpa_supplicant-${device}.conf
}

wifi_activate() {
    local device=$1
    local configfile=/etc/wpa_supplicant/wpa_supplicant-${device}.conf

    if [ -f $configfile ]
    then
        systemctl enable wpa_supplicant@${device}
        systemctl start wpa_supplicant@${device}
    fi
}
