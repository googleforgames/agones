# v{version}

This is the {version} release of Agones.

Check the [README](https://github.com/googleforgames/agones/tree/release-{version}) for details on features, installation and usage.

**Implemented enhancements:**

{ insert enhancements from the changelog and/or security and breaking changes }

{ if release candidate }
Documentation: https://development.agones.dev/site/
{ end }

See [CHANGELOG](https://github.com/googleforgames/agones/blob/release-{version}/CHANGELOG.md) for more details on changes.

Images available with this release:

- [gcr.io/agones-images/agones-controller:{version}](https://gcr.io/agones-images/agones-controller:{version})
- [gcr.io/agones-images/agones-sdk:{version}](https://gcr.io/agones-images/agones-sdk:{version})
- [gcr.io/agones-images/agones-ping:{version}](https://gcr.io/agones-images/agones-ping:{version})
- [gcr.io/agones-images/agones-allocator:{version}](https://gcr.io/agones-images/agones-allocator:{version})
- [gcr.io/agones-images/cpp-simple-server:{example-version}](https://gcr.io/agones-images/cpp-simple-server:{example-version})
- [gcr.io/agones-images/nodejs-simple-server:{example-version}](https://gcr.io/agones-images/nodejs-simple-server:{example-version})
- [gcr.io/agones-images/rust-simple-server:{example-version}](https://gcr.io/agones-images/rust-simple-server:{example-version})
- [gcr.io/agones-images/unity-simple-server:{example-version}](https://gcr.io/agones-images/unity-simple-server:{example-version})
- [gcr.io/agones-images/udp-server:{example-version}](https://gcr.io/agones-images/udp-server:{example-version})
- [gcr.io/agones-images/tcp-server:{example-version}](https://gcr.io/agones-images/tcp-server:{example-version})
- [gcr.io/agones-images/xonotic-example:{example-version}](https://gcr.io/agones-images/xonotic-example:{example-version})
- [gcr.io/agones-images/supertuxkart-example:{example-version}](https://gcr.io/agones-images/supertuxkart-example:{example-version})
- [gcr.io/agones-images/crd-client:{example-version}](https://gcr.io/agones-images/crd-client:{example-version})

Helm chart available with this release:

- [`helm install agones/agones --version {version}`](https://agones.dev/chart/stable/agones-{version}.tgz)

> Make sure to add our stable helm repository using `helm repo add agones https://agones.dev/chart/stable`
