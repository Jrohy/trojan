#!/bin/bash
# source: https://github.com/trojan-gfw/trojan-quickstart
set -eo pipefail

# trojan: 0, trojan-go: 1
TYPE=0

INSTALL_VERSION=""

while [[ $# > 0 ]];do
    KEY="$1"
    case $KEY in
        -v|--version)
        INSTALL_VERSION="$2"
        echo -e "prepare install $INSTALL_VERSION version..\n"
        shift
        ;;
        -g|--go)
        TYPE=1
        ;;
        *)
                # unknown option
        ;;
    esac
    shift # past argument or value
done
#############################

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

ARCH=$(uname -m 2> /dev/null)
if [[ $ARCH != x86_64 && $ARCH != aarch64 ]];then
    echo "not support $ARCH machine".
    exit 1
fi
if [[ $TYPE == 0 && $ARCH != x86_64 ]];then
    echo "trojan not support $ARCH machine"
    exit 1
fi

if [[ $TYPE == 0 ]];then
    CHECKVERSION="https://api.github.com/repos/trojan-gfw/trojan/releases/latest"
else
    CHECKVERSION="https://api.github.com/repos/p4gefau1t/trojan-go/releases"
fi
NAME=trojan
if [[ -z $INSTALL_VERSION ]];then
    VERSION=$(curl -H 'Cache-Control: no-cache' -s "$CHECKVERSION" | grep 'tag_name' | cut -d\" -f4 | sed 's/v//g' | head -n 1)
else
    if [[ -z `curl -H 'Cache-Control: no-cache' -s "$CHECKVERSION"|grep 'tag_name'|grep $INSTALL_VERSION` ]];then
        echo "no $INSTALL_VERSION version file!"
        exit 1
    fi
    VERSION=`echo "$INSTALL_VERSION"|sed 's/v//g'`
fi
if [[ $TYPE == 0 ]];then
    TARBALL="$NAME-$VERSION-linux-amd64.tar.xz"
    DOWNLOADURL="https://github.com/trojan-gfw/$NAME/releases/download/v$VERSION/$TARBALL"
else
    [[ $ARCH == x86_64 ]] && TARBALL="trojan-go-linux-amd64.zip" || TARBALL="trojan-go-linux-armv8.zip" 
    DOWNLOADURL="https://github.com/p4gefau1t/trojan-go/releases/download/v$VERSION/$TARBALL"
fi

TMPDIR="$(mktemp -d)"
INSTALLPREFIX="/usr/bin/$NAME"
SYSTEMDPREFIX=/etc/systemd/system

BINARYPATH="$INSTALLPREFIX/$NAME"
CONFIGPATH="/usr/local/etc/$NAME/config.json"
SYSTEMDPATH="$SYSTEMDPREFIX/$NAME.service"

echo Creating $NAME install directory
mkdir -p $INSTALLPREFIX /usr/local/etc/$NAME

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
    cat > "$CONFIGPATH" << EOF
{
    "run_type": "server",
    "local_addr": "0.0.0.0",
    "local_port": 443,
    "remote_addr": "127.0.0.1",
    "remote_port": 80,
    "password": [
        "password1",
        "password2"
    ],
    "log_level": 1,
    "ssl": {
        "cert": "/path/to/certificate.crt",
        "key": "/path/to/private.key",
        "key_password": "",
        "cipher": "ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384",
        "cipher_tls13": "TLS_AES_128_GCM_SHA256:TLS_CHACHA20_POLY1305_SHA256:TLS_AES_256_GCM_SHA384",
        "prefer_server_cipher": true,
        "alpn": [
            "http/1.1"
        ],
        "alpn_port_override": {
            "h2": 81
        },
        "reuse_session": true,
        "session_ticket": false,
        "session_timeout": 600,
        "plain_http_response": "",
        "curves": "",
        "dhparam": ""
    },
    "tcp": {
        "prefer_ipv4": false,
        "no_delay": true,
        "keep_alive": true,
        "reuse_port": false,
        "fast_open": false,
        "fast_open_qlen": 20
    },
    "mysql": {
        "enabled": false,
        "server_addr": "127.0.0.1",
        "server_port": 3306,
        "database": "trojan",
        "username": "trojan",
        "password": "",
        "key": "",
        "cert": "",
        "ca": ""
    }
}
EOF
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