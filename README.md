# Conveyor

Conveyor builds Docker images. Fast.

## How it works

1. Conveyor receives a build request via a GitHub commit webhook.
2. Conveyor builds and tags the resulting image with 3 tags: `latest`, the git commit sha and the git branch.
3. It then pushes the image to the Docker registry and adds a commit status to the GitHub commit.

![](https://s3.amazonaws.com/ejholmes.github.com/U21Pu.png)

## Installation

1. Conveyor needs access to pull GitHub repositories. The easiest way to do this is to add a bot user to your organization and generate an ssh key for them. Once you've done that, create a new S3 bucket and upload `id_rsa` and `id_rsa.pub` to the root of the bucket.
2. Create a new CloudFormation stack using [cloudformation.json](./cloudformation.json) in this repo.

## Configuration

The server command has the following available options:

```
NAME:
   server - Run an http server to build Docker images whenever a push event happens on GitHub

USAGE:
   command server [command options] [arguments...]

OPTIONS:
   --port '8080'        Port to run the server on [$PORT]
   --github.token         GitHub API token to use when updating commit statuses on repositories. [$GITHUB_TOKEN]
   --github.secret        Shared secret used by GitHub to sign webhook payloads. This secret will be used to verify that the request came from GitHub. [$GITHUB_SECRET]
   --dry          Enable dry run mode. [$DRY]
   --builder.image 'remind101/conveyor-builder' A docker image to use to perform the build. [$BUILDER_IMAGE]
   --logger 'stdout://'       The logger to use. Available options are `stdout://`, or `s3://bucket`. [$LOGGER]
   
```

## Performance

Conveyor is designed to be faster than alternative build systems like the Docker Hub or Quay. It does this by making the following tradeoffs.

1. It uses the latest version of Docker (1.8), which has a number of performance improvements when building and pushing images.
2. It pulls the last built image for the branch to maximize the number of layers that can be used from the cache.

## Cache

By default, conveyor will pull the last built image for the branch. This isn't always desirable, so you can disable the initial `docker pull` by adding the following to the git commit description:

```
[docker nocache]
```

## Scale Out

Conveyor supports two methods to scale out to multiple machines.

### Docker Swarm

The first method to scale out Conveyor is to scale out using [Docker Swarm](https://github.com/docker/swarm). Using this method, Conveyor runs its builds across a cluster of Docker daemons. The advantage of using this method is that you don't need to provide a `queue` flag since Conveyor can use an in memory queue.

### Queue

The recommended way to scale out is to scale out using a build queue. Using this method, you run the `conveyor worker` subcommand on a machine that hosts a local Docker daemon. The worker process will pull build requests off of the queue and perform the build. The `conveyor server` command can then run completely separate from the worker nodes.

![](https://dl.dropboxusercontent.com/u/1906634/GitHub/Conveyor%20-%20Split.png)

Conveyor currently supports the following build queues:

1. SQS

## API (Soon)

Conveyor also sports a restful API for triggering builds. You can use this with tooling to, say for example, trigger a build before you deploy.

### POST /builds

This endpoint will create a build and stream it's output back to the client.

**Example Request**

```json
{
  "Repository": "remind101/acme-inc",
  "Sha": "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57"
}
```

**Example Command**

```console
$ conveyor build remind101/acme-inc:827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57
remind101/acme-inc@sha256:44be4f298f764a2a8eb7eecce5383d2b
```

## Development

First, bootstrap the `remind101/conveyor-builder` image, SSH keys and docker config:

```console
$ make bootstrap
```

Then start it up with docker-compose:

```console
$ docker-compose up
```

If you want to test external GitHub webhooks, the easiest way to do that is using ngrok:

```console
$ ngrok $(docker-machine ip default):8080
```

Then add a new `push` webhook to a repo, pointed at the ngrok URL. No secret is necessary unless you set `GITHUB_SECRET` in `.env`.
