FROM ubuntu:14.04

RUN apt-get update
RUN apt-get install -y curl git wget
RUN wget -qO- https://get.docker.com/ | sh
ADD ./bin/install /bin/install
RUN /bin/install
ADD ./bin/build /bin/build

VOLUME /var/lib/docker

ENTRYPOINT ["build"]
