
# Abuse string lengths to convert a netmask to a CIDR number
#
# Input: A netmask in decimal, like "255.255.192.0"
# Output: The number of 1 bits at the beginning of the mask
mask2cdr() {
    local mask=$1
    # These spaces are significant
    local mask_lookup="0   128 192 224 240 248 252 254 "

    # The number of characters in the string "255." is 4, and the number of
    # bits in the octet 255 is 8.  So let's just take the 255's off the
    # beginning of the netmask and calculate the number of bits in the 255s
    # that way
    local mask_size="${#mask}"
    local mask_trimmed="${mask##*255.}"
	local mask_trimmed_size=${#mask_trimmed}
    local mask_major_size=$(( (mask_size - mask_trimmed_size) * 2 ))

    # The "last significant octet" is the last octet that isn't either a 255 or
    # a 0. Or it might be a 0, if the mask consists of only 255s and 0s.
    local mask_last_significant_octet=${mask_trimmed%%.*}

    # Trim everything from the lookup after the LSO
    local mask_lookup_less_lso=${mask_lookup%%${mask_last_significant_octet}*}
    local mask_place_in_lookup=${#mask_lookup_less_lso}
    local mask_minor_size=$(( mask_place_in_lookup / 4))

    echo $(( mask_major_size + mask_minor_size ))
}

# Convert a CIDR number to a netmask
#
# Input: Number of bits in the netmask
# Output: The actual netmask
#
# Rather more straightforward than going the other way round
cidr2mask() {
    local cidr=$1
    local mask=""
    local mask_lookup=(0 128 192 224 240 248 252 254)
    local octets=4

    while ((cidr >= 8))
    do
        mask="255.$mask"
        ((cidr -= 8))
        ((octets --))
    done

    if ((octets > 0))
    then
        mask="$mask${mask_lookup[$cidr]}"
        ((octets -= 1))
    fi

    while ((octets --> 0))
    do
        mask="${mask}.0"
    done

    echo "$mask"
}

