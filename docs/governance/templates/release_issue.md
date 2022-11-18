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
- [ ] Ensure the next RC and stable releases in the Google Calendar have the correct version number.
- [ ] Ensure the next version milestone is created.
- [ ] Any issues in the current milestone that are not closed, move to next milestone.
- [ ] If release candidate add the label `feature-freeze-do-not-merge` to any feature pull requests.
- [ ] `git checkout main && git pull --rebase upstream main`
- [ ] If full release, run `make site-deploy SERVICE={version}-1`, (replace . with -)
   - For example, if you are creating the 1.18.0 release, then you would deploy the 1-17-0 service (release minus one, and then replace dots with dashes).
- [ ] Run `make gen-changelog` to generate the CHANGELOG.md (if release candidate 
  `make gen-changelog RELEASE_VERSION={version}-rc`). You will need your 
  [GitHub Personal Access Token](https://github.com/settings/tokens) for this.
- [ ] Ensure the [helm `tag` value][values] is correct (should be {version} if a full release, {version}-rc if release candidate)
- [ ] Ensure the [helm `Chart` version values][chart] are correct (should be {version} if a full release, {version}-rc if release candidate)
- [ ] Update SDK Package Versions
    - [ ] Update the package version in [`sdks/nodejs/package.json`][package.json] and [`sdks/nodejs/package-lock.json`][package-lock.json] by running `npm version {version}` if a full release or `npm version {version}-rc` if release candidate
    - [ ] Ensure the [`sdks/csharp/sdk/AgonesSDK.nuspec` and `sdks/csharp/sdk/csharp-sdk.csproj`][csharp] versions are correct (should be {version} if a full release, {version}-rc if release candidate)
    - [ ] Update the package version in the [`sdks/unity/package.json`][unity] package file's `Version` field to {version} if a full release, {version}-rc if release candidate
- [ ] Run `make gen-install`
- [ ] Run `make test-examples-on-gcr` to ensure all example images exist on gcr.io/agones-images-
- [ ] Create a *draft* release with the [release template][release-template]
  - [ ] Make a `tag` with the release version.
- [ ] Site updated
  - [ ] Copy the draft release content into a new `/site/content/en/blog/releases` content (this will be what you send via email). 
  - [ ] Review all `link_test` and `data-proofer-ignore` attributes and remove for link testing
  - [ ] If full release, review and remove all instances of the `feature` shortcode
  - [ ] If full release, add a link to previous version's documentation to nav dropdown.
  - [ ] config.toml updates:
    - [ ] If full release, update `release_branch` to the new release branch for {version}.
    - [ ] If full release, update `release-version` with the new release version {version}.
    - [ ] If full release, copy `dev_supported_k8s` to `supported_k8s`.
    - [ ] If full release, copy `dev_aks_minor_supported_k8s` to `aks_minor_supported_k8s`.
    - [ ] If full release, copy `dev_minikube_minor_supported_k8s` to `minikube_minor_supported_k8s`.
    - [ ] If full release, update documentation with updated example images tags.
- [ ] Create PR with these changes, and merge them with an approval.
- [ ] Run `git remote update && git checkout main && git reset --hard upstream/main` to ensure your code is in line 
   with upstream  (unless this is a hotfix, then do the same, but for the release branch)
- [ ] Publish SDK packages
   - [ ] Run `make sdk-shell-node` to get interactive shell to publish node package. Requires Google internal process
     to publish.
   - [ ] Run `make sdk-publish-csharp` to deploy to NuGet. Requires login credentials. (if release candidate: 
   `make sdk-publish-csharp RELEASE_VERSION={version}-rc`).
   Will need [NuGet API Key](https://www.nuget.org/account/apikeys) from Agones account.
- [ ] Run `make do-release`. (if release candidate: `make do-release RELEASE_VERSION={version}-rc`) to create and push the docker images and helm chart.
- [ ] Run `make shell` and run `gcloud config configurations activate <your development project>` to switch Agones
    development tooling off of the `agones-images` project.
- [ ] Do a `helm repo add agones https://agones.dev/chart/stable` / `helm repo update` and verify that the new
 version is available via the command `helm search repo agones --versions --devel`.
- [ ] Do a `helm install --namespace=agones-system agones agones/agones` 
    (`helm install --namespace=agones-system agones agones/agones --version={version}-rc` if release candidate) and a smoke test to confirm everything is working.
- [ ] Attach all assets found in the `release` folder to the draft GitHub Release.
- [ ] If release candidate check the pre-release box on the draft GitHub Release
- [ ] Copy any review changes from the release blog post into the draft GitHub release.
- [ ] Publish the draft GitHub Release.
- [ ] Email the [mailing list][list] with the release details (copy-paste the release blog post).
- [ ] Paste the announcement blog post to the #users Slack group.
- [ ] Post to the [agonesdev](https://twitter.com/agonesdev) Twitter account.
- [ ] If full release, run `git checkout main`.
- [ ] If full release, then increment the `base_version` in [`build/Makefile`][build-makefile]
- [ ] If full release move [helm `tag` value][values] is set to {version}+1-dev
- [ ] If full release move the [helm `Chart` version values][chart] is to {version}+1-dev
- [ ] If full release, change to the `sdks/nodejs` directory and run the command `npm version {version}+1-dev` to update the package version
- [ ] If full release move the [`sdks/csharp/sdk/AgonesSDK.nuspec` and `sdks/csharp/sdk/csharp-sdk.csproj`][csharp] to {version}+1-dev
- [ ] If full release update the [`sdks/unity/package.json`][unity] package file's `Version` field to {version}+1-dev
- [ ] If full release, remove `feature-freeze-do-not-merge` labels from all pull requests
- [ ] Run `make gen-install gen-api-docs`
- [ ] Create PR with these changes, and merge them with approval
- [ ] Close this issue.
- [ ] If full release, close the current milestone. *Congratulations!* - the release is now complete! :tada: :clap: :smile: :+1:

[values]: https://github.com/googleforgames/agones/blob/main/install/helm/agones/values.yaml#L33
[chart]: https://github.com/googleforgames/agones/blob/main/install/helm/agones/Chart.yaml
[list]: https://groups.google.com/forum/#!forum/agones-discuss
[release-template]: https://github.com/googleforgames/agones/blob/main/docs/governance/templates/release.md
[build-makefile]: https://github.com/googleforgames/agones/blob/main/build/Makefile
[package.json]: https://github.com/googleforgames/agones/blob/main/sdks/nodejs/package.json
[package-lock.json]: https://github.com/googleforgames/agones/blob/main/sdks/nodejs/package-lock.json
[csharp]: https://github.com/googleforgames/agones/blob/main/sdks/csharp/sdk/
