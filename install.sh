
docker run -d --name test --net=host --privileged trojan-manager /sbin/init

docker run -it -d --name test -v /var/run/docker.sock:/var/run/docker.sock -v `which docker`:`which docker` centos tail -f