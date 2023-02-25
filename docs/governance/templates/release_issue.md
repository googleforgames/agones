# Release {version}

<!--
This is the release issue template. Make a copy of the markdown in this page
and copy it into a release issue. Fill in relevant values, found inside {}

*** VERSION SHOULD BE IN THE FORMAT OF 1.x.x NOT v1.x.x ***
!-->

## Prerequisites

- [ ] Have at least `Editor` level access to `agones-images` project.
- [ ] Have permission to publish new versions of the App Engine application.
- [ ] Have write access to Agones GitHub repository.
- [ ] Run `git remote -v` and see:
    - [ ] An `origin` remote that points to a personal fork of Agones, such as `git@github.com:yourname/agones.git`.
    - [ ] An `upstream` remote that points to `git@github.com:googleforgames/agones.git`.
- [ ] Have a [GitHub Personal Access Token](https://github.com/settings/tokens) with repo permissions.
- [ ] Have a `gcloud config configurations` configuration called `agones-images` that points to the same project.
- [ ] Edit access to the [Agones Release Calendar](https://calendar.google.com/calendar/u/0?cid=Z29vZ2xlLmNvbV84MjhuOGYxOGhmYnRyczR2dTRoMXNrczIxOEBncm91cC5jYWxlbmRhci5nb29nbGUuY29t)

## Steps

- [ ] Run `make shell` and run `gcloud config configurations activate agones-images`.
- [ ] Review [closed issues with no milestone](https://github.com/googleforgames/agones/issues?q=is%3Aissue+is%3Aclosed+no%3Amilestone++-label%3Ainvalid+-label%3Aduplicate+-label%3Aquestion+-label%3Awontfix++-label%3Aarea%2Fmeta) and add relevant ones to the current milestone.
    - Issues tagged as `invalid`, `duplicate`, `question`, `wontfix`, or `area/meta` can be ignored
- [ ] Review closed issues in the current milestone to ensure that they have appropriate tags.
- [ ] Review [merged PRs that have no milestone](https://github.com/googleforgames/agones/pulls?q=is%3Apr+is%3Amerged+no%3Amilestone+) and add them to the current milestone.
- [ ] Review merged PRs in the current milestone to ensure that they have appropriate tags.
- [ ] Ensure the next stable releases in the Google Calendar have the correct version number.
- [ ] Ensure the next version milestone is created.
- [ ] Any issues in the current milestone that are not closed, move to next milestone.
- [ ] `git checkout main && git pull --rebase upstream main`
- [ ] Run `make release-deploy-site`
- [ ] Run `make build-release` to generate the CHANGELOG.md and to ensure all example images exist on us-docker.pkg.dev/agones-images/examples.
- [ ] Download all the artifacts from the cloud build.
- [ ] Move the CHANGELOG.md to the root of this repository, replacing any previous versions.
- [ ] Ensure the [helm `tag` value][values] is correct (should be {version})
- [ ] Ensure the [helm `Chart` version values][chart] are correct (should be {version})
- [ ] Update SDK Package Versions
  - [ ] Update the package version in [`sdks/nodejs/package.json`][package.json] and [`sdks/nodejs/package-lock.json`][package-lock.json] by running `npm version {version}`
  - [ ] Ensure the [`sdks/csharp/sdk/AgonesSDK.nuspec` and `sdks/csharp/sdk/csharp-sdk.csproj`][csharp] versions are correct (should be {version})
  - [ ] Update the package version in the [`sdks/unity/package.json`][unity] package file's `Version` field to {version}
- [ ] Run `make gen-install`
- [ ] Create a *draft* release with the [release template][release-template]
    - [ ] Make a `tag` with the release version.
- [ ] Site updated
  - [ ] Copy the draft release content into a new `/site/content/en/blog/releases` content (this will be what you send via email).
  - [ ] In `site/content/en/docs/Installation/_index.md #agones-and-kubernetes-supported-versions`, for the current version, replace `{{% k8s-version %}}` with hardcoded Kubernetes versions supported by the current version. And add a row for the Agones release version with `{{% k8s-version %}}` as its supported Kubernetes versions.
  - [ ] Review all `link_test` and `data-proofer-ignore` attributes and remove for link testing
  - [ ] Review and remove all instances of the `feature` shortcode
  - [ ] Add a link to previous version's documentation to nav dropdown.
  - [ ] config.toml updates:
    - [ ] Update `release_branch` to the new release branch for {version}.
    - [ ] Update `release-version` with the new release version {version}.
    - [ ] Copy `dev_supported_k8s` to `supported_k8s`.
    - [ ] Copy `dev_k8s_api_version` to `k8s_api_version`.
    - [ ] Copy `dev_gke_example_cluster_version` to `gke_example_cluster_version`.
    - [ ] Copy `dev_aks_example_cluster_version` to `aks_example_cluster_version`.
    - [ ] Copy `dev_eks_example_cluster_version` to `eks_example_cluster_version`.
    - [ ] Copy `dev_minikube_example_cluster_version` to `minikube_example_cluster_version`.
    - [ ] Update documentation with updated example images tags.
- [ ] Create PR with these changes, and merge them with an approval.
- [ ] Run `git remote update && git checkout main && git reset --hard upstream/main` to ensure your code is in line
  with upstream  (unless this is a hotfix, then do the same, but for the release branch)
- [ ] Publish SDK packages
    - [ ] Run `make sdk-shell-node` to get interactive shell to publish node package. Requires Google internal process
      to publish.
    - [ ] Run `make sdk-publish-csharp` to deploy to NuGet. Requires login credentials.
      Will need [NuGet API Key](https://www.nuget.org/account/apikeys) from Agones account.
- [ ] Run `make do-release` to create and push the docker images and helm chart.
- [ ] Run `make shell` and run `gcloud config configurations activate <your development project>` to switch Agones
    development tooling off of the `agones-images` project.
- [ ] Do a `helm repo add agones https://agones.dev/chart/stable` / `helm repo update` and verify that the new
  version is available via the command `helm search repo agones --versions --devel`.
- [ ] Do a `helm install --namespace=agones-system agones agones/agones`
   and a smoke test to confirm everything is working.
- [ ] Attach all assets found in the `release` folder to the draft GitHub Release.
- [ ] Copy any review changes from the release blog post into the draft GitHub release.
- [ ] Publish the draft GitHub Release.
- [ ] Email the [mailing list][list] with the release details (copy-paste the release blog post).
- [ ] Paste the announcement blog post to the #users Slack group.
- [ ] Post to the [agonesdev](https://twitter.com/agonesdev) Twitter account.
- [ ] Run `git checkout main`.
- [ ] Then increment the `base_version` in [`build/Makefile`][build-makefile]
- [ ] Move [helm `tag` value][values] is set to {version}+1-dev
- [ ] Move the [helm `Chart` version values][chart] is to {version}+1-dev
- [ ] Change to the `sdks/nodejs` directory and run the command `npm version {version}+1-dev` to update the package version
- [ ] Move the [`sdks/csharp/sdk/AgonesSDK.nuspec` and `sdks/csharp/sdk/csharp-sdk.csproj`][csharp] to {version}+1-dev
- [ ] Update the [`sdks/unity/package.json`][unity] package file's `Version` field to {version}+1-dev
- [ ] Run `make gen-install gen-api-docs`
- [ ] Create PR with these changes, and merge them with approval
- [ ] Close this issue.
- [ ] Close the current milestone. _Congratulations!_ - the release is now complete! :tada: :clap: :smile: :+1:

[values]: https://github.com/googleforgames/agones/blob/main/install/helm/agones/values.yaml#L33
[chart]: https://github.com/googleforgames/agones/blob/main/install/helm/agones/Chart.yaml
[list]: https://groups.google.com/forum/#!forum/agones-discuss
[release-template]: https://github.com/googleforgames/agones/blob/main/docs/governance/templates/release.md
[build-makefile]: https://github.com/googleforgames/agones/blob/main/build/Makefile
[package.json]: https://github.com/googleforgames/agones/blob/main/sdks/nodejs/package.json
[package-lock.json]: https://github.com/googleforgames/agones/blob/main/sdks/nodejs/package-lock.json
[csharp]: https://github.com/googleforgames/agones/blob/main/sdks/csharp/sdk/
