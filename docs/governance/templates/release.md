# v{version}

This is the {version} release of Agones.

{ write description of release }

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
  { run `make release-example-image-markdown` to populate example images section below (will be more in output than in example) }
- [us-docker.pkg.dev/agones-images/examples/cpp-simple-server:{example-version}](https://us-docker.pkg.dev/agones-images/examples/cpp-simple-server:{example-version})
- [us-docker.pkg.dev/agones-images/examples/crd-client:{example-version}](https://us-docker.pkg.dev/agones-images/examples/crd-client:{example-version})
- [us-docker.pkg.dev/agones-images/examples/nodejs-simple-server:{example-version}](https://us-docker.pkg.dev/agones-images/examples/nodejs-simple-server:{example-version})
- [us-docker.pkg.dev/agones-images/examples/rust-simple-server:{example-version}](https://us-docker.pkg.dev/agones-images/examples/rust-simple-server:{example-version})

Helm chart available with this release:

- <a href="https://agones.dev/chart/stable/agones-{version}.tgz" data-proofer-ignore>
  <code>helm install agones agones/agones --version {version}</code></a>

> Make sure to add our stable helm repository using `helm repo add agones https://agones.dev/chart/stable`
