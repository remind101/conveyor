# Docker Builder

This a [Builder](../../builder) implementation that builds Docker images inside Docker. So meta.

The official Docker image to perform the builds is [remind101/conveyor-builder](https://github.com/remind101/conveyor-builder) but you can provide any Docker image that you want, if you want to add custom logic to your builds.

The following environment variables are provided to the Docker image when it's run:

Environment Variable | Description | Example
---------------------|-------------|--------
`REPOSITORY` | The name of the repository | `remind101/acme-inc`
`SHA` | The git commit sha that should be built | `827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57`
`BRANCH` | The branch that the build was triggered from. This value is optional and may not be present | `master`
`DRY` | Set to `true` if this should be considered a "dry" run. It's up to the Docker image to determine what this means, but with the official image it will perform a build but not push to the registry. | `true` or ``
`CACHE` | Determines whether caching should be enabled on this build. The official image will pull an image tagged with the branch if this is set. | `on` or `off`

In addition, it will attach any volumes from a container named `data`, which you can use to add any secrets like a `.dockercfg` or ssh keys.
