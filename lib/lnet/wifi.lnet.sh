#!/bin/bash

# Runs the "iwlist" utility on the device to scan for wireless APs
#
# Since iwlist has some wildly-inconsistent output, every single line of
# its output has to be parsed separately.  This means that this is
# probably REALLY fragile, because if the output was that fragile to
# begine with, it's definitely subject to the whims of the developers of
# iwlist.

wifi_find_aps() {
    local device=$1

    iwlist $device scan | awk '
        BEGIN {
        }

        # Cell 23 - Address: E2:B4:F7:99:56:C9
        /Cell/ { 
            if(cell_essid) {
                print "WIFI_AP[" cell_num "]=" cell_essid
                print "WIFI_QUALITY[" cell_num "]=" cell_quality
                print "WIFI_ENCRYPTION[" cell_num "]=" cell_encryption
            }
            cell_num = $2
        }

        # Quality=25/70  Signal level=-85 dBm

        /Quality/ {
            OFS=FS
            cell_quality=$1
            FS="="
            $0=cell_quality
            cell_quality=$2
            FS=OFS
        }

        # Encryption key:on

        /Encryption/ {
            if($2 == "key:on") {
                cell_encryption="true"
            } else {
                cell_encryption="false"
            }
        }
        
        # ESSID:"xg100n-98598d-1"

        /ESSID/ { 
            OFS=FS
            FS=":"
            $0=$0 # force resplit of input
            cell_essid=$2
            FS=OFS
        }
        
        END { # get the last one
            if(cell_essid) {
                print "WIFI_APS[" cell_num "]=" cell_essid
                print "WIFI_QUALITY[" cell_num "]=" cell_quality
            }
        }'
}

# wifi_ap_password <AP> <password>
#
# Creates a wpa_supplicant fragment
wifi_ap_password() {
    local device=$1
    local AP=$2
    local PASS=$3

    wpa_passphrase "$AP" "$PASS" >> /etc/wpa_supplicant/wpa_supplicant-${device}.conf
    chmod 600 /etc/wpa_supplicant/wpa_supplicant-${device}.conf
}
