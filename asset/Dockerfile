FROM centos:7

LABEL maintainer "Jrohy <euvkzx@Jrohy.com>"

ENV VERSION_CHECK "https://api.github.com/repos/Jrohy/trojan/releases/latest"

ENV DOWNLAOD_URL "https://github.com/Jrohy/trojan/releases/download/"

RUN yum install socat bash-completion iptables crontabs openssl -y && \
    LATEST_VERSION=`curl -H 'Cache-Control: no-cache' -s "$VERSION_CHECK" | grep 'tag_name' | cut -d\" -f4` && \
    curl -L "$DOWNLAOD_URL/$LATEST_VERSION/trojan" -o /usr/local/bin/trojan && \
    chmod +x /usr/local/bin/trojan && \
    echo "source <(trojan completion bash)" >> ~/.bashrc && \
    touch /.run.log && \
    yum clean all

ENTRYPOINT ["tail", "-f", "/.run.log"]