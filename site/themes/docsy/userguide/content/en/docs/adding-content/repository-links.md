---
title: Repository Links
weight: 9
description: Help your users interact with your source repository.
---

The Docsy [docs and blog layouts](/docs/adding-content/content/#adding-docs-and-blog-posts) include links for readers to edit the page or create issues for your docs or project via your site's source repository. The current generated links for each docs or blog page are:

* **View page source**: Brings the user to the page source in your docs repo.
* **Edit this page**: Brings the user to an editable version of the page content in their fork (if available) of your docs repo. If the user doesn't have a current fork of your docs repo, they are invited to create one before making their edit. The user can then create a pull request for your docs.
* **Create child page**: Brings the user to a create new file form in their fork of your docs repo.  The new file will be located as a child of the page they clicked the link on.  The form will be pre-populated with a template the user can edit to create their page.  You can change this by adding `assets/stubs/new-page-template.md` to your own project.
* **Create documentation issue**: Brings the user to a new issue form in your docs repo with the name of the current page as the issue's title.
* **Create project issue** (optional): Brings the user to a new issue form in your project repo. This can be useful if you have separate project and docs repos and your users want to file issues against the project feature being discussed rather than your docs.

This page shows you how to configure these links.

Currently, Docsy supports only GitHub repository links "out of the box". Since GitLab can handle the same link scheme, it should work as well. If you are using another repository such as Bitbucket and would like generated repository links, feel free to [add a feature request or update our theme](/docs/contribution-guidelines/).

## Link configuration

There are four variables you can configure in `hugo.toml`/`hugo.yaml`/`hugo.json` to set up links, as well as one in your page metadata.

### `github_repo`

The URL for your site's source repository. This is used to generate the **Edit this page**, **Create child page**, and **Create documentation issue** links.

{{< tabpane persistLang=false >}}
{{< tab header="Configuration file:" disabled=true />}}
{{< tab header="hugo.toml" lang="toml" >}}
github_repo = "https://github.com/google/docsy"
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
github_repo: 'https://github.com/google/docsy'
{{< /tab >}}
{{< tab header="hugo.json" lang="json" >}}
{
  "github_repo": "https://github.com/google/docsy"
}
{{< /tab >}}
{{< /tabpane >}}

### `github_subdir` (optional)

Specify a value here if your content directory is not in your repo's root directory. For example, this site is in the `userguide` subdirectory of its repo. Setting this value means that your edit links will go to the right page.

{{< tabpane persistLang=false >}}
{{< tab header="Configuration file:" disabled=true />}}
{{< tab header="hugo.toml" lang="toml" >}}
github_subdir = "userguide"
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
github_subdir: 'userguide'
{{< /tab >}}
{{< tab header="hugo.json" lang="json" >}}
{
  "github_subdir": "userguide"
}
{{< /tab >}}
{{< /tabpane >}}

### `github_project_repo` (optional)

Specify a value here if you have a separate project repo and you'd like your users to be able to create issues against your project from the relevant docs. The **Create project issue** link appears only if this is set.

{{< tabpane persistLang=false >}}
{{< tab header="Configuration file:" disabled=true />}}
{{< tab header="hugo.toml" lang="toml" >}}
github_project_repo = "https://github.com/google/docsy"
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
github_project_repo: 'https://github.com/google/docsy'
{{< /tab >}}
{{< tab header="hugo.json" lang="json" >}}
{
  "github_project_repo": "https://github.com/google/docsy"
}
{{< /tab >}}
{{< /tabpane >}}

### `github_branch` (optional)

Specify a value here if you have would like to reference a different branch for the other github settings like **Edit this page** or **Create project issue**.

{{< tabpane persistLang=false >}}
{{< tab header="Configuration file:" disabled=true />}}
{{< tab header="hugo.toml" lang="toml" >}}
github_branch = "release"
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
github_branch: 'release'
{{< /tab >}}
{{< tab header="hugo.json" lang="json" >}}
{
  "github_branch": "release"
}
{{< /tab >}}
{{< /tabpane >}}

### `path_base_for_github_subdir` (optional)

Suppose that the source files for all of the pages under `content/some-section`
come from another repo, such as a [git submodule][]. Add settings like these to
the **section's index page** so that the repository links for all pages in that
section refer to the originating repo:

{{< tabpane persistLang=false >}}
{{< tab header="Front matter:" disabled=true />}}
{{< tab header="toml" lang="toml" >}}
+++
title = "Some super section"
[cascade]
github_repo = "https://github.com/some-username/another-repo/"
github_subdir = "docs"
path_base_for_github_subdir = "content/some-section"
…
+++
{{< /tab >}}
{{< tab header="yaml" lang="yaml" >}}
---
title: Some super section
cascade:
  github_repo: https://github.com/some-username/another-repo/
  github_subdir: docs
  path_base_for_github_subdir: content/some-section
…
---
{{< /tab >}}
{{< tab header="json" lang="json" >}}
{
  "title": "Some super section",
  "cascade": {
    "github_repo": "https://github.com/some-username/another-repo/",
    "github_subdir": "docs",
    "path_base_for_github_subdir": "content/some-section"
  }
}
{{< /tab >}}
{{< /tabpane >}}

As an example, consider a page at the path
`content/some-section/subpath/some-page.md` with `github_branch` globally set to
`main`. The index page settings above will generate the following edit link for
`some-page.md`:

```nocode
https://github.com/some-username/another-repo/edit/main/docs/subpath/some-page.md
```

If you only have a single page originating from another repo, then omit the
`cascade` key and write, at the top-level, the same settings as illustrated
above.

If you'd like users to create project issues in the originating repo as well,
then also set `github_project_repo`, something like this:

```yaml
---
...
cascade:
  github_repo: &repo https://github.com/some-username/another-repo/
  github_project_repo: *repo
...
---
```

{{< tabpane persistLang=false >}}
{{< tab header="Front matter:" disabled=true />}}
{{< tab header="toml" lang="toml" >}}
+++
…
[cascade]
github_repo = "https://github.com/some-username/another-repo/"
github_project_repo = "https://github.com/some-username/another-repo/"
…
+++
{{< /tab >}}
{{< tab header="yaml" lang="yaml" >}}
---
…
cascade:
  github_repo: &repo https://github.com/some-username/another-repo/
  github_project_repo: *repo
…
---
{{< /tab >}}
{{< tab header="json" lang="json" >}}
{
  "cascade": {
    "github_repo": "https://github.com/some-username/another-repo/",
    "github_project_repo": "https://github.com/some-username/another-repo/"
  }
}
{{< /tab >}}
{{< /tabpane >}}

{{% alert title="Tip" %}}
Please note that the YAML code fragment makes use of [Yaml anchor](https://support.atlassian.com/bitbucket-cloud/docs/yaml-anchors/). Use of Yaml anchors is optional, but it helps keep the settings [DRY](https://en.wikipedia.org/wiki/Don%27t_repeat_yourself).
{{% /alert %}}

The `path_base_for_github_subdir` setting is a regular expression, so you can
use it even if you have a site with [multiple languages][] for example:

{{< tabpane persistLang=false >}}
{{< tab header="Front matter:" disabled=true />}}
{{< tab header="toml" lang="toml" >}}
+++
…
path_base_for_github_subdir = "content/\w+/some-section"
…
+++
{{< /tab >}}
{{< tab header="yaml" lang="yaml" >}}
---
…
path_base_for_github_subdir: content/\w+/some-section
…
---
{{< /tab >}}
{{< tab header="json" lang="json" >}}
{
…
  "path_base_for_github_subdir": "content/\w+/some-section"
…
}
{{< /tab >}}
{{< /tabpane >}}

In situations where a page originates from a file under a different name, you
can specify `from` and `to` path-rename settings. Here's an example where an
index file is named `README.md` in the originating repo:

{{< tabpane persistLang=false >}}
{{< tab header="Front matter:" disabled=true />}}
{{< tab header="toml" lang="toml" >}}
+++
…
github_repo = "https://github.com/some-username/another-repo/"
github_subdir = "docs"

[path_base_for_github_subdir]
from = "content/some-section/(.*?)/_index.md"
to = "$1/README.md"
…
+++
{{< /tab >}}
{{< tab header="yaml" lang="yaml" >}}
---
…
github_repo: https://github.com/some-username/another-repo/
github_subdir: docs
path_base_for_github_subdir:
  from: content/some-section/(.*?)/_index.md
  to: $1/README.md
…
---
{{< /tab >}}
{{< tab header="json" lang="json" >}}
{
  …
  "github_repo": "https://github.com/some-username/another-repo/",
  "github_subdir": "docs",
  "path_base_for_github_subdir": {
    "from": "content/some-section/(.*?)/_index.md",
    "to": "$1/README.md"
  },
  …
}
{{< /tab >}}
{{< /tabpane >}}

### `github_url` (optional)

{{% alert title="Deprecation note" color="warning" %}}
  This setting is deprecated. Use [path_base_for_github_subdir][] instead.

  [path_base_for_github_subdir]: #path_base_for_github_subdir-optional
{{% /alert %}}

Specify a value for this **in your page metadata** to set a specific edit URL for this page, as in the following example:

{{< tabpane persistLang=false >}}
{{< tab header="Front matter:" disabled=true />}}
{{< tab header="toml" lang="toml" >}}
+++
title = "Some page"
github_url = "https://github.com/some-username/another-repo/edit/main/README.md"
…
+++
{{< /tab >}}
{{< tab header="yaml" lang="yaml" >}}
---
title: Some page
github_url: https://github.com/some-username/another-repo/edit/main/README.md
…
---
{{< /tab >}}
{{< tab header="json" lang="json" >}}
{
  "title": "Some page",
  "github_url": "https://github.com/some-username/another-repo/edit/main/README.md",
  …
}
{{< /tab >}}
{{< /tabpane >}}

This can be useful if you have page source files in multiple Git repositories,
or require a non-GitHub URL. Pages using this value have **Edit this page**
links only.

## Disabling links

You can use CSS to selectively disable (hide) links. For example, add the
following to your [projects's `_styles_project.scss`][project-style-files] file
to hide **Create child page** links from all pages:

```scss
.td-page-meta--child { display: none !important; }
```

Each link kind has an associated unique class named `.td-page-meta--KIND`, as
defined by the following table:

Link kind | Class name
--- | ---
View page source | `.td-page-meta--view`
Edit this page | `.td-page-meta--edit`
Create child page | `.td-page-meta--child`
Create documentation issue | `.td-page-meta--issue`
Create project issue | `.td-page-meta--project-issue`

Of course, you can also use these classes to give repository links unique styles
for your project.

[git submodule]: https://git-scm.com/book/en/v2/Git-Tools-Submodules
[multiple languages]: {{< relref "language" >}}
[project-style-files]: {{< relref "lookandfeel#project-style-files" >}}
