FROM ubuntu:14.04

# Let's start with some basic stuff.
RUN apt-get update && apt-get install -qqy \
    apt-transport-https \
    ca-certificates \
    curl \
    git \
    wget \
    lxc \
    linux-image-extra-virtual \
    iptables
RUN curl -sSL https://get.docker.com/ubuntu/ | sh
ADD ./bin/build /bin/build
ADD ./bin/wrapdocker /bin/wrapdocker

VOLUME /var/lib/docker
ENTRYPOINT ["build"]
