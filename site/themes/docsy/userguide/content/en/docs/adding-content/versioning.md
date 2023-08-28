---
title: "Doc Versioning"
date: 2020-02-02
weight: 4
description: >
   Customize navigation and banners for multiple versions of your docs.
---

Depending on your project's releases and versioning, you may want to let your
users access previous versions of your documentation. How you deploy the
previous versions is up to you. This page describes the Docsy features that you
can use to provide navigation between the various versions of your docs and
to display an information banner on the archived sites.

## Adding a version drop-down menu

If you add some `[params.versions]` in `hugo.toml`/`hugo.yaml`/`hugo.json`, the Docsy theme adds a
version selector drop down to the top-level menu. You specify a URL and a name
for each version you would like to add to the menu, as in the following example:

{{< tabpane persistLang=false >}}
{{< tab header="Configuration file:" disabled=true />}}
{{< tab header="hugo.toml" lang="toml" >}}
# Add your release versions here
[[params.versions]]
  version = "master"
  url = "https://master.kubeflow.org"

[[params.versions]]
  version = "v0.2"
  url = "https://v0-2.kubeflow.org"

[[params.versions]]
  version = "v0.3"
  url = "https://v0-3.kubeflow.org"
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
params:
  versions:
    - version: master
      url: 'https://master.kubeflow.org'
    - version: v0.2
      url: 'https://v0-2.kubeflow.org'
    - version: v0.3
      url: 'https://v0-3.kubeflow.org'
{{< /tab >}}
{{< tab header="hugo.json" lang="json" >}}
{
  "params": {
    "versions": [
      {
        "version": "master",
        "url": "https://master.kubeflow.org"
      },
      {
        "version": "v0.2",
        "url": "https://v0-2.kubeflow.org"
      },
      {
        "version": "v0.3",
        "url": "https://v0-3.kubeflow.org"
      }
    ]
  }
}
{{< /tab >}}
{{< /tabpane >}}

Remember to add your current version so that users can navigate back!

The default title for the version drop-down menu is **Releases**. To change the
title, change the `version_menu` parameter in `hugo.toml`/`hugo.yaml`/`hugo.json`:

{{< tabpane persistLang=false >}}
{{< tab header="Configuration file:" disabled=true />}}
{{< tab header="hugo.toml" lang="toml" >}}
version_menu = "Releases"
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
version_menu: 'Releases'
{{< /tab >}}
{{< tab header="hugo.json" lang="json" >}}
"version_menu": "Releases"
{{< /tab >}}
{{< /tabpane >}}

If you set the `version_menu_pagelinks` parameter to `true`, then links in the version drop-down menu
point to the current page in the other version, instead of the main page.
This can be useful if the document doesn't change much between the different versions.
Note that if the current page doesn't exist in the other version, the link will be broken.

You can read more about Docsy menus in the guide to
[navigation and search](/docs/adding-content/navigation/).

## Displaying a banner on archived doc sites

If you create archived snapshots for older versions of your docs, you can add a
note at the top of every page in the archived docs to let readers know that
they’re seeing an unmaintained snapshot and give them a link to the latest
version.

For example, see the archived docs for
[Kubeflow v0.6](https://v0-6.kubeflow.org/docs/):

<figure>
  <img src="/images/version-banner.png"
       alt="A text box explaining that this is an unmaintained snapshot of the docs."
       class="mt-3 mb-3 border border-info rounded" />
  <figcaption>Figure 1. The banner on the archived docs for Kubeflow v0.6
  </figcaption>
</figure>

To add the banner to your doc site, make the following changes in your
`hugo.toml`/`hugo.yaml`/`hugo.json` file:

1. Set the `archived_version` parameter to `true`:

    {{< tabpane persistLang=false >}}
{{< tab header="Configuration file:" disabled=true />}}
{{< tab header="hugo.toml" lang="toml" >}}
archived_version = true
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
archived_version: true
{{< /tab >}}
{{< tab header="hugo.json" lang="json" >}}
"archived_version": true
{{< /tab >}}
    {{< /tabpane >}}

1. Set the `version` parameter to the version of the archived doc set. For
  example, if the archived docs are for version 0.1:

    {{< tabpane persistLang=false >}}
{{< tab header="Configuration file:" disabled=true />}}
{{< tab header="hugo.toml" lang="toml" >}}
version = "0.1"
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
version: '0.1'
{{< /tab >}}
{{< tab header="hugo.json" lang="json" >}}
"version": "0.1"
{{< /tab >}}
    {{< /tabpane >}}

1. Make sure that `url_latest_version` contains the URL of the website that you
  want to point readers to. In most cases, this should be the URL of the latest
  version of your docs:

    {{< tabpane persistLang=false >}}
{{< tab header="Configuration file:" disabled=true />}}
{{< tab header="hugo.toml" lang="toml" >}}
url_latest_version = "https://your-latest-doc-site.com"
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
url_latest_version: 'https://your-latest-doc-site.com'
{{< /tab >}}
{{< tab header="hugo.json" lang="json" >}}
"url_latest_version": "https://your-latest-doc-site.com"
{{< /tab >}}
    {{< /tabpane >}}
