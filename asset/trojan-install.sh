#!/bin/bash
# source: https://github.com/trojan-gfw/trojan-quickstart
set -eo pipefail

# trojan: 0, trojan-go: 1
TYPE=0

[[ $1 == "go" ]] && TYPE=1

function prompt() {
    while true; do
        read -p "$1 [y/N] " yn
        case $yn in
            [Yy] ) return 0;;
            [Nn]|"" ) return 1;;
        esac
    done
}

if [[ $(id -u) != 0 ]]; then
    echo Please run this script as root.
    exit 1
fi

if [[ $(uname -m 2> /dev/null) != x86_64 ]]; then
    echo Please run this script on x86_64 machine.
    exit 1
fi

if [[ $TYPE == 0 ]];then
    CHECKVERSION="https://api.github.com/repos/trojan-gfw/trojan/releases/latest"
else
    CHECKVERSION="https://api.github.com/repos/p4gefau1t/trojan-go/releases"
fi
NAME=trojan
VERSION=$(curl -H 'Cache-Control: no-cache' -s "$CHECKVERSION" | grep 'tag_name' | cut -d\" -f4 | sed 's/v//g' | head -n 1)
if [[ $TYPE == 0 ]];then
    TARBALL="$NAME-$VERSION-linux-amd64.tar.xz"
    DOWNLOADURL="https://github.com/trojan-gfw/$NAME/releases/download/v$VERSION/$TARBALL"
else
    TARBALL="trojan-go-linux-amd64.zip"
    DOWNLOADURL="https://github.com/p4gefau1t/trojan-go/releases/download/v$VERSION/$TARBALL"
fi

TMPDIR="$(mktemp -d)"
INSTALLPREFIX="/usr/bin/$NAME"
SYSTEMDPREFIX=/etc/systemd/system

BINARYPATH="$INSTALLPREFIX/$NAME"
CONFIGPATH="/usr/local/etc/$NAME/config.json"
SYSTEMDPATH="$SYSTEMDPREFIX/$NAME.service"

echo Creating $NAME install directory
mkdir -p $INSTALLPREFIX

echo Entering temp directory $TMPDIR...
cd "$TMPDIR"

echo Downloading $NAME $VERSION...
curl -LO --progress-bar "$DOWNLOADURL" || wget -q --show-progress "$DOWNLOADURL"

echo Unpacking $NAME $VERSION...
if [[ $TYPE == 0 ]];then
    tar xf "$TARBALL"
    cd "$NAME"
else
    if [[ -z `command -v unzip` ]];then
        if [[ `command -v dnf` ]];then
            dnf install unzip -y
        elif [[ `command -v yum` ]];then
            yum install unzip -y
        elif [[ `command -v apt-get` ]];then
            apt-get install unzip -y
        fi
    fi
    unzip "$TARBALL"
    mv trojan-go trojan
fi

echo Installing $NAME $VERSION to $BINARYPATH...
install -Dm755 "$NAME" "$BINARYPATH"

echo Installing $NAME server config to $CONFIGPATH...
if ! [[ -f "$CONFIGPATH" ]] || prompt "The server config already exists in $CONFIGPATH, overwrite?"; then
    if [[ $TYPE == 0 ]];then
        install -Dm644 examples/server.json-example "$CONFIGPATH"
    else
        install -Dm644 example/server.json "$CONFIGPATH"
    fi
else
    echo Skipping installing $NAME server config...
fi

if [[ -d "$SYSTEMDPREFIX" ]]; then
    echo Installing $NAME systemd service to $SYSTEMDPATH...
    [[ $TYPE == 1 ]] && { NAME="trojan-go"; FLAG="-config"; }
    cat > "$SYSTEMDPATH" << EOF
[Unit]
Description=$NAME
After=network.target network-online.target nss-lookup.target mysql.service mariadb.service mysqld.service

[Service]
Type=simple
StandardError=journal
ExecStart=$BINARYPATH $FLAG $CONFIGPATH
ExecReload=/bin/kill -HUP \$MAINPID
Restart=on-failure
RestartSec=3s

[Install]
WantedBy=multi-user.target
EOF
    echo Reloading systemd daemon...
    systemctl daemon-reload
fi

echo Deleting temp directory $TMPDIR...
rm -rf "$TMPDIR"

echo Done!