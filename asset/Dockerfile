FROM centos:7

LABEL maintainer="Jrohy <euvkzx@Jrohy.com>"

ARG VERSION_CHECK="https://api.github.com/repos/Jrohy/trojan/releases/latest"

ARG DOWNLAOD_URL="https://github.com/Jrohy/trojan/releases/download/"

ARG SERVICE_URL="https://raw.githubusercontent.com/Jrohy/trojan/master/asset/trojan-web.service"

RUN yum install socat bash-completion crontabs iptables openssl unzip -y && \
    LATEST_VERSION=`curl -H 'Cache-Control: no-cache' -s "$VERSION_CHECK" | grep 'tag_name' | cut -d\" -f4` && \
    curl -L "$DOWNLAOD_URL/$LATEST_VERSION/trojan" -o /usr/local/bin/trojan && \
    curl -L $SERVICE_URL -o /etc/systemd/system/trojan-web.service && \
    chmod +x /usr/local/bin/trojan && \
    echo "source <(trojan completion bash)" >> ~/.bashrc && \
    yum clean all
