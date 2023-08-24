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
- [ ] Have a `gcloud config configurations` configuration called `agones-images` that points to the same project.
- [ ] Edit access to the [Agones Release Calendar](https://calendar.google.com/calendar/u/0?cid=Z29vZ2xlLmNvbV84MjhuOGYxOGhmYnRyczR2dTRoMXNrczIxOEBncm91cC5jYWxlbmRhci5nb29nbGUuY29t)
- [ ] Get approval for [Release Title and Description](https://docs.google.com/document/d/1bRZCxYB8lrVcrru41b6s5D_9uU0zS49vVGdBhg0yDIY/edit)

## Steps

- [ ] Run `make shell` and run `gcloud config configurations activate agones-images`.
- [ ] Review merged PRs in the current milestone to ensure that they have appropriate tags.
- [ ] `git checkout main && git pull --rebase upstream main`
- [ ] Run `make pre-build-release` to ensure all example images exist on agones-images/examples repository and to deploy the {version}-1 service on GCP/App Engine/Services.
- [ ] Run `make sdk-update-version release_stage=before version={version}` file. This command will update the version number in the sdks/install files to {version}.
- [ ] Create a _draft_ release with the [release template][release-template].
  - run `make release-example-image-markdown` to populate example images and append the output in `Images available with this release` section
  - [ ] Draft a new release with [release draft][release-draft]. Update the `Tag version` and `Release title` with the release version and click on `Generate release notes` to generate the release notes with `Full Changelog` info for {version}. Make sure to add the description. Include the `Images available with this release` section from the previous step that will be present after the `Full Changelog` and save the draft.
  - [ ] copy the {version} release details from the `Full Changelog` and paste it on top of the CHANGELOG.md file
- [ ] Site updated
  - [ ] Create a new file named {version}.md in `/site/content/en/blog/releases`. Copy the draft release content in this file (this will be what you send via email)[refer the previous release file].
  - run `make site-server` frequently to make sure everything looks fine for the release in your localhost
  - [ ] In `site/content/en/docs/Installation/_index.md #agones-and-kubernetes-supported-versions`, for the current version, replace `{{% k8s-version %}}` with hardcoded Kubernetes versions supported by the current version. And add a row for the Agones release version with `{{% k8s-version %}}` as its supported Kubernetes versions.
  - [ ] Run `make del-data-proofer-ignore FILENAME={version}-1.md` to remove `data-proofer-ignore` attribute from previous release blog. Review all occurrences of the link_test and data-proofer-ignore attributes globally. Exclude html and release files.
  - [ ] Run `make feature-shortcode-update version={version}` to remove all instances of the `feature expiryVersion` shortcode, including the associated content, while preserving the rest of the content within the .md files located in site/content/en/docs. Additionally, ensure that only the block of `feature publishVersion` is removed without affecting the content.
  - [ ] Add a link to previous version's documentation to nav dropdown in `site/layouts/partials/navbar.html` on top and Run `make update-navbar-version FILENAME=site/layouts/partials/navbar.html` to remove the older version from the dropdown list.
  - [ ] config.toml updates:
    - [ ] Run `make site-config-update-version` to update the release version and sync data between dev and prod.
    - [ ] Update documentation with updated example images tags.
- [ ] Create PR with these changes, and merge them with an approval.
- [ ] Run `git remote update && git checkout main && git reset --hard upstream/main` to ensure your code is in line
      with upstream (unless this is a hotfix, then do the same, but for the release branch)
- [ ] Publish SDK packages
  - [ ] Run `make sdk-shell-node` to get interactive shell to publish node package. Requires Google internal process
        to publish.
  - [ ] Run `make sdk-publish-csharp` to deploy to NuGet. Requires login credentials.
        Will need [NuGet API Key](https://www.nuget.org/account/apikeys) from Agones account.
- [ ] Run `make post-build-release` to build the artifacts in GCS(These files will be attached in the release notes) and to push the latest images in the release repository and push chart on agones-chart.
- [ ] Run `make shell` and run `gcloud config configurations activate <your development project>` to switch Agones
      development tooling off of the `agones-images` project.
- [ ] Do a `helm repo add agones https://agones.dev/chart/stable` / `helm repo update` and verify that the new
      version is available via the command `helm search repo agones --versions --devel`.
- [ ] Do a `helm install --namespace=agones-system agones agones/agones` or `helm install --create-namespace --namespace=agones-system agones agones/agones` if the namespace was deleted and a smoke test to confirm everything is working.
- [ ] Attach all assets found in the cloud storage with {version} to the draft GitHub Release.
- [ ] Copy any review changes from the release blog post into the draft GitHub release.
- [ ] Publish the draft GitHub Release.
- [ ] Run `make release-branch` to create a release branch and run `gcloud config configurations activate <your development project>` to switch Agones development tooling off of the `agones-images` project.
- [ ] Email mailing lists with the release details (copy-paste the release blog post). Refer to the [Internal Mailing list posting guide][Internal Mailing list posting guide] for details. 
- [ ] Paste the announcement blog post to the #users Slack group.
- [ ] Post to the [agonesdev](https://twitter.com/agonesdev) Twitter account.
- [ ] Run `git checkout main`.
- [ ] Run `make sdk-update-version release_stage=after version={version}` file. This command will update the version number in the sdks/install files to {version}+1-dev.
- [ ] Run `make sdks-rust-cargo-version-update`. Commit the changes in a feature branch called `prep/{version}+1`. 
- [ ] Run `make sdk-publish-rust`. This command executes `cargo login` for authentication, performs a dry-run publish, and if that succeeds, does the actual publish. Will need [crate's API TOKEN](https://crates.io/settings/tokens) from your crate's account.
- [ ] Create PR with these changes, and merge them with approval
- [ ] Close this issue. _Congratulations!_ - the release is now complete! :tada: :clap: :smile: :+1:

[release-template]: https://github.com/googleforgames/agones/blob/main/docs/governance/templates/release.md
[release-draft]: https://github.com/googleforgames/agones/releases
[build-makefile]: https://github.com/googleforgames/agones/blob/main/build/Makefile
[Internal Mailing list posting guide]: https://docs.google.com/document/d/1qYR9ccVURgujqFAIpjpSN2GRcCeQ29ow5H_V4sm4RGs/edit#heading=h.zge9gjrt8ws8
