FROM ubuntu:14.04

# Let's start with some basic stuff.
RUN apt-get update && apt-get install -qqy \
    apt-transport-https \
    ca-certificates \
    curl \
    git \
    wget
RUN curl -sSL https://get.docker.com | sh
ADD ./bin/build /bin/build
ADD ./bin/wrapdocker /bin/wrapdocker

# Log docker daemon logs to a file
ENV LOG file

VOLUME /var/lib/docker
ENTRYPOINT ["build"]
