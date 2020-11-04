# v{version}

This is the {version} release of Agones.

Check the <a href="https://github.com/googleforgames/agones/tree/release-{version}" data-proofer-ignore>README</a> for details on features, installation and usage.

**Implemented enhancements:**

{ insert enhancements from the changelog and/or security and breaking changes }

{ if release candidate }
Documentation: https://development.agones.dev/site/
{ end }

See <a href="https://github.com/googleforgames/agones/blob/release-{version}/CHANGELOG.md" data-proofer-ignore>CHANGELOG</a> for more details on changes.

Images available with this release:

- [gcr.io/agones-images/agones-controller:{version}](https://gcr.io/agones-images/agones-controller:{version})
- [gcr.io/agones-images/agones-sdk:{version}](https://gcr.io/agones-images/agones-sdk:{version})
- [gcr.io/agones-images/agones-ping:{version}](https://gcr.io/agones-images/agones-ping:{version})
- [gcr.io/agones-images/agones-allocator:{version}](https://gcr.io/agones-images/agones-allocator:{version})
- [gcr.io/agones-images/cpp-simple-server:{example-version}](https://gcr.io/agones-images/cpp-simple-server:{example-version})
- [gcr.io/agones-images/crd-client:{example-version}](https://gcr.io/agones-images/crd-client:{example-version})
- [gcr.io/agones-images/nodejs-simple-server:{example-version}](https://gcr.io/agones-images/nodejs-simple-server:{example-version})
- [gcr.io/agones-images/rust-simple-server:{example-version}](https://gcr.io/agones-images/rust-simple-server:{example-version})
- [gcr.io/agones-images/simple-game-server:{example-version}](https://gcr.io/agones-images/simple-game-server:{example-version})
- [gcr.io/agones-images/supertuxkart-example:{example-version}](https://gcr.io/agones-images/supertuxkart-example:{example-version})
- [gcr.io/agones-images/tcp-server:{example-version}](https://gcr.io/agones-images/tcp-server:{example-version})
- [gcr.io/agones-images/udp-server:{example-version}](https://gcr.io/agones-images/udp-server:{example-version})
- [gcr.io/agones-images/unity-simple-server:{example-version}](https://gcr.io/agones-images/unity-simple-server:{example-version})
- [gcr.io/agones-images/xonotic-example:{example-version}](https://gcr.io/agones-images/xonotic-example:{example-version})

Helm chart available with this release:

- <a href="https://agones.dev/chart/stable/agones-{version}.tgz" data-proofer-ignore>
  <code>helm install agones agones/agones --version {version}</code></a>

> Make sure to add our stable helm repository using `helm repo add agones https://agones.dev/chart/stable`
