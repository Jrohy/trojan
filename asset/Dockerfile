FROM --platform=$TARGETPLATFORM centos:7

LABEL maintainer="Jrohy <euvkzx@Jrohy.com>"

ARG TARGETARCH

ARG VERSION_CHECK="https://api.github.com/repos/Jrohy/trojan/releases/latest"

ARG DOWNLAOD_URL="https://github.com/Jrohy/trojan/releases/download/"

ARG SERVICE_URL="https://raw.githubusercontent.com/Jrohy/trojan/master/asset/trojan-web.service"

ARG SYSTEMCTL_URL="https://raw.githubusercontent.com/gdraheim/docker-systemctl-replacement/master/files/docker/systemctl.py"

RUN yum install socat bash-completion crontabs iptables openssl unzip -y && \
    LATEST_VERSION=`curl -H 'Cache-Control: no-cache' -s "$VERSION_CHECK" | grep 'tag_name' | cut -d\" -f4` && \
    [[ $TARGETARCH =~ "arm64" ]] && ARCH="arm64" || ARCH="amd64" && \
    curl -fL "$DOWNLAOD_URL/$LATEST_VERSION/trojan-linux-$ARCH" -o /usr/local/bin/trojan && \
    curl -L $SERVICE_URL -o /etc/systemd/system/trojan-web.service && \
    curl -L $SYSTEMCTL_URL -o /usr/bin/systemctl && \
    chmod +x /usr/local/bin/trojan /usr/bin/systemctl && \
    echo "source <(trojan completion bash)" >> ~/.bashrc && \
    yum clean all
