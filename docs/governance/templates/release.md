# v{version}

This is the {version} release of Agones.

Check the [README](https://github.com/GoogleCloudPlatform/agones/tree/{release-branch}) for details on features, installation and usage.

Features in this release:

{ insert enhancements from the changelog }

See [CHANGELOG](https://github.com/GoogleCloudPlatform/agones/blob/{release-branch}/CHANGELOG.md) for more details on changes.

This software is currently alpha, and subject to change. Not to be used in production systems.

Images available with this release:

- [gcr.io/agones-images/agones-controller:{version}](https://gcr.io/agones-images/agones-controller:{version})
- [gcr.io/agones-images/agones-sdk:{version}](https://gcr.io/agones-images/agones-sdk:{version})
- [gcr.io/agones-images/cpp-simple-server:{example-version}](https://gcr.io/agones-images/cpp-simple-server:{example-version})
- [gcr.io/agones-images/udp-server:{example-version}](https://gcr.io/agones-images/udp-server:{example-version})
- [gcr.io/agones-images/xonotic-example:{example-version}](https://gcr.io/agones-images/xonotic-example:{example-version})

Helm chart available with this release:

- [`helm install agones/agones --version {version}`](https://agones.dev/chart/stable/agones-{version}.tgz)

> Make sure to add our stable helm repository using `helm repo add https://agones.dev/chart/stable`
