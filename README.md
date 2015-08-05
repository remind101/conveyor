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

Conveyor only needs to talk to the docker daemon API. The easiest way to scale out is to scale out Docker using [Docker Swarm](https://github.com/docker/swarm).

## API (Soon)

Conveyor also sports a restful API for triggering builds. You can use this with tooling to, say for example, trigger a build before you deploy.

### POST /builds

This endpoint will create a build and stream it's output back to the client.

**Example Request**

```json
{
  "Repository": "remind101/acme-inc",
  "Sha": "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57",
  "Branch": "master"
}
```

## Development

First, cp `.env.sample` to `.env` and add values in the environment variables. The `GITHUB_TOKEN` needs the `repo:status` scope.

```console
cp .env.sample .env
```

Then start it up with docker-compose:

```console
$ docker-compose up
```

If you want to test external GitHub webhooks, the easiest way to do that is using ngrok:

```console
$ ngrok 8080
```
