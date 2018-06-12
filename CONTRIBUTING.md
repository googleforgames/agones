# How to Contribute

We'd love to accept your patches and contributions to this project. There are
just a few small guidelines you need to follow.

## Contributor License Agreement

Contributions to this project must be accompanied by a Contributor License
Agreement. You (or your employer) retain the copyright to your contribution,
this simply gives us permission to use and redistribute your contributions as
part of the project. Head over to <https://cla.developers.google.com/> to see
your current agreements on file or to sign a new one.

You generally only need to submit a CLA once, so if you've already submitted one
(even if it was for a different project), you probably don't need to do it
again.

## Code of Conduct

Participation in this project comes under the [Contributor Covenant Code of Conduct](code-of-conduct.md)

## Submitting code via Pull Requests

- We follow the [Github Pull Request Model](https://help.github.com/articles/about-pull-requests/) for
  all contributions.
- For large bodies of work, we recommend creating an issue and labelling it
  "[kind/design](https://github.com/googleprivate/agones/issues?q=is%3Aissue+is%3Aopen+label%3Akind%2Fdesign)"
  outlining the feature that you wish to build, and describing how it will be implemented. This gives a chance
  for review to happen early, and ensures no wasted effort occurs.
- All submissions, including submissions by project members, will require review before being merged.
- Once review has occurred, please rebase your PR down to a single commit. This will ensure a nice clean Git history.
- Finally - *Thanks* for considering submitting code to Agones!

## Formatting

When submitting pull requests, make sure to do the following:

- Format all Go code with [gofmt](https://golang.org/cmd/gofmt/). Many people
  use [goimports](https://godoc.org/golang.org/x/tools/cmd/goimports) which
  fixes import statements and formats code in the same style of `gofmt`.
- Remove trailing whitespace. Many editors will do this automatically.
- Ensure any new files have [a trailing newline](https://stackoverflow.com/questions/5813311/no-newline-at-end-of-file)

## Continuous Integration

Continuous integration is provided by [Google Cloud Container Builder](https://cloud.google.com/container-builder/),
through the [cloudbuilder.yaml](./cloudbuild.yaml) file found at the root of the directory.

Build success/failure with relevant details are pushed automatically to pull requests via the not (yet 😉) opensourced
build system.

See the [Container Builder documentation](https://cloud.google.com/container-builder/docs/) for more details on
how to edit and expand the build process.

### Additional Resources

#### Extending Kubernetes

- [Kubernetes Custom Resources](https://kubernetes.io/docs/concepts/api-extension/custom-resources/) -
  This is how we define our own resource names (`GameServer`, etc) within Kubernetes.
- [Extend the Kubernetes API with CustomResourceDefinitions](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/) -
  This page shows how to install a custom resource into the Kubernetes API by creating a CustomResourceDefinition.
- [Joe Beda's TGIK Controller](https://github.com/jbeda/tgik-controller) -
  [Joe Beda](https://twitter.com/jbeda) did a video series on writing controllers for Kubernetes.
  **This is the best resource for learning about controllers and Kubernetes.**
- [Kubernetes Sample Controller](https://github.com/kubernetes/sample-controller) -
  Example of a Custom Resources with a Kubernetes Controller.
- [Kubernetes Code Generator](https://github.com/kubernetes/code-generator) -
  The tooling that generated the Go libraries for the Custom Resource we define
- [Kubernetes Controller Best Practices](https://github.com/kubernetes/community/blob/master/contributors/devel/controllers.md) -
  Set of best practices written for writing Controllers inside Kubernetes. Also a great list for everywhere else too.
- [Writing Kube Controllers for Everyone - Maciej Szulik, Red Hat](https://www.youtube.com/watch?v=AUNPLQVxvmw) -
  A great intro video into coding for Controllers, and explaining Informers and Listers.
- [@markmandel](https://github.com/markmandel) regularly streams his development of Agones on [Twitch](https://www.twitch.tv/markmandel).
  You can find the full archive on [YouTube](https://www.youtube.com/playlist?list=PLqqp1QEhKwa5aNivDIE4SS21ehE9Zt0VZ)
  

#### Coding and Development

- [How to write a good Git Commit message](https://chris.beams.io/posts/git-commit/) -
  Great way to make sure your Pull Requests get accepted.
