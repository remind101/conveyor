FROM ubuntu:14.04

# Let's start with some basic stuff.
RUN apt-get update && apt-get install -qqy \
    apt-transport-https \
    ca-certificates \
    curl \
    git \
    wget
RUN apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D
RUN mkdir -p /etc/apt/sources.list.d && \
    echo deb https://apt.dockerproject.org/repo ubuntu-trusty main > /etc/apt/sources.list.d/docker.list
RUN apt-get update && apt-get install -q -y \
    docker-engine=1.8.1-0~trusty
ADD ./bin/build /bin/build
ADD ./bin/wrapdocker /bin/wrapdocker

# Log docker daemon logs to a file
ENV LOG file

VOLUME /var/lib/docker
ENTRYPOINT ["build"]
