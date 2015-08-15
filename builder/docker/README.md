# Builder

This a Docker image suitable for using as a builder for conveyor. It performs the following actions:

1. Clones the GitHub repo.
2. Pulls that last built docker image for the given branch.
3. Builds a new image.
4. Tags the new image with `latest` as well as the name of the branch and the git sha.
5. Pushes the image to the docker registry.

## Usage

This image expects that the following files to be present:

* `/var/run/conveyor/.dockercfg`: .dockercfg file for credentials to use when pulling.
* `/var/run/conveyor/.ssh`: Should container an `id_rsa` and `id_rsa.pub` file which gives access to pull GitHub repos.
