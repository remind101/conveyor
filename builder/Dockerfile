FROM ubuntu:14.04

RUN apt-get update
RUN apt-get install -y curl git wget
RUN curl -sSL https://test.docker.com | sh
ADD ./bin/install /bin/install
RUN /bin/install
ADD ./bin/build /bin/build
ADD ./bin/wrapdocker /bin/wrapdocker

VOLUME /var/lib/docker
ENTRYPOINT ["build"]
