
msgbox() {
    if [ -z "$3" ]
    then
        H=10
    else
        H=$3
    fi
    $DIALOG --title "$1" --msgbox "$2" $H 50
}

inputbox() {
    $DIALOG --nocancel --inputbox "$1" 0 0 "$2"
}

confirm() {
    $DIALOG $2 --yesno "$1" 8 50
}

