# Conveyor

Conveyor is a build system for Docker containers as fast as possible.

## How it works

1. Conveyor receives a build request via a GitHub commit webhook.
2. Conveyor builds and tags the resulting image with 3 tags: `latest`, the git commit sha and the git branch.
3. It then pushes the image to the Docker registry.

## Performance

Conveyor is designed to be faster than alternative build systems like the Docker Hub or Quay. It does this by making the following tradeoffs.

1. It _should_ run on a single machine so that it can keep most of the Docker images in cache.
2. It uses the latest version of Docker 1.8, which has a number of performance improvements when building and pushing images.
3. It pulls the last built image for the branch to maximize the number of layers that can be used from the cache.
