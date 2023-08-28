---
title: "Create a new site: Start a new site from scratch"
linkTitle: "Start a site from scratch"
date: 2021-12-08T09:21:54+01:00
weight: 3
description: >
  Create a new Hugo site from scratch with Docsy as a Hugo Module
---

The simplest approach to creating a Docsy site is [copying our example site](/docs/get-started/docsy-as-module/example-site-as-template/). However, if you're an experienced Hugo user or the site structure of our example site doesn't meet your needs, you may prefer to create a new site from scratch. With this option, you'll get Docsy look and feel, navigation, and other features, but you'll need to specify your own site structure.

These instructions give you a minimum file structure for your site project only, so that you build and extend your actual site step by step. The first step is adding the Docsy theme as a [Hugo Module](https://gohugo.io/hugo-modules/) to your site. If needed, you can easily [update](/docs/updating/) the module to the latest revision from the Docsy GitHub repo.

## TL;DR: Setup for the impatient expert

At your command prompt, run the following:

{{< tabpane >}}
{{< tab header="CLI:" disabled=true />}}
{{< tab header="Unix shell"  lang="Bash" >}}
hugo new site my-new-site
cd  my-new-site
hugo mod init github.com/me/my-new-site
hugo mod get github.com/google/docsy@v{{% param "version" %}}
cat >> hugo.toml <<EOL
[module]
proxy = "direct"
[[module.imports]]
path = "github.com/google/docsy"
[[module.imports]]
path = "github.com/google/docsy/dependencies"
EOL
hugo server
{{< /tab >}}
{{< tab header="Windows command line" lang="Batchfile" >}}
hugo new site my-new-site
cd  my-new-site
hugo mod init github.com/me/my-new-site
hugo mod get github.com/google/docsy@v{{% param "version" %}}
(echo [module]^

proxy = "direct"^

[[module.imports]]^

path = "github.com/google/docsy"^

[[module.imports]]^

path = "github.com/google/docsy/dependencies")>>hugo.toml
hugo server
{{< /tab >}}
{{< /tabpane >}}


You now can preview your new site inside your browser at [http://localhost:1313](http://localhost:1313/).

## Detailed Setup instructions

Specifying the [Docsy theme](https://github.com/google/docsy) as Hugo Module for your minimal site gives you all the theme-y goodness, but you'll need to specify your own site structure.

### Create your new skeleton project

To create a new Hugo site project and then add the Docs theme as a submodule, run the following commands from your project's root directory.

```bash
hugo new site my-new-site
cd  my-new-site
```

This will create a minimal site structure, containing the folders `archetypes`, `content`, `data`, `layouts`, `static`, and `themes` and a configuration file (default: `hugo.toml`).

{{% alert title="Tip" %}}
In Hugo 0.110.0 the default config base filename was changed to `hugo.toml`.
If you are using hugo 0.110 or above, consider renaming your `config.toml` to `hugo.toml`!
{{% /alert %}}

### Import the Docsy theme module as a dependency of your site

Only sites that are Hugo Modules themselves can import other modules. To turn your site into a Hugo Module, run the following commands in your newly created site directory:

```bash
hugo mod init github.com/me/my-new-site
```

This creates two new files, `go.mod` for the module definitions and `go.sum` which holds the checksums for module verification.

Next declare the Docsy theme module as a dependency for your site.

```bash
hugo mod get github.com/google/docsy@v{{% param "version" %}}
```

This command adds the `docsy` theme module to your definition file `go.mod`.

### Add theme module configuration settings

Add the settings in the following snippet at the end of your site's [configuration file] (default: `hugo.toml`) and save the file.

{{< tabpane >}}
{{< tab header="Configuration file:" disabled=true />}}
{{< tab header="hugo.toml"  lang="toml" >}}
[module]
  proxy = "direct"
  # uncomment line below for temporary local development of module
  # replacements = "github.com/google/docsy -> ../../docsy"
  [module.hugoVersion]
    extended = true
    min = "0.73.0"
  [[module.imports]]
    path = "github.com/google/docsy"
    disable = false
  [[module.imports]]
    path = "github.com/google/docsy/dependencies"
    disable = false
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
module:
  proxy: direct
  hugoVersion:
    extended: true
    min: 0.73.0
  imports:
    - path: github.com/google/docsy
      disable: false
    - path: github.com/google/docsy/dependencies
      disable: false
{{< /tab >}}
{{< tab header="hugo.json"  lang="json" >}}
{
  "module": {
    "proxy": "direct",
    "hugoVersion": {
      "extended": true,
      "min": "0.73.0"
    },
    "imports": [
      {
        "path": "github.com/google/docsy",
        "disable": false
      },
      {
        "path": "github.com/google/docsy/dependencies",
        "disable": false
      }
    ]
  }
}
{{< /tab >}}
{{< /tabpane >}}

You can find details of what these configuration settings do in the [Hugo modules documentation](https://gohugo.io/hugo-modules/configuration/#module-config-top-level).
Depending on your environment you may need to tweak them slightly, for example by adding a proxy to use when downloading remote modules.

### Preview your site

To build and preview your site locally:

```bash
hugo server
```

By default, your site will be available at [http://localhost:1313](http://localhost:1313/). When encountering problems, have a look at the [known issues](/docs/get-started/known_issues/#macos) on MacOS.

You may get Hugo errors for missing parameters and values when you try to build your site. This is usually because you're missing default values for some configuration settings that Docsy uses - once you add them your site should build correctly. You can find out how to add configuration in [Basic site configuration](/docs/get-started/basic-configuration/) - we recommend copying the example site configuration even if you're creating a site from scratch as it provides defaults for many required configuration parameters.

## What's next?

* Add some [basic configuration](/docs/get-started/basic-configuration/)
* [Add content and customize your site](/docs/adding-content/)
* Get some ideas from our [Example Site](https://github.com/google/docsy-example) and other [Examples](/docs/examples/).
* [Publish your site](/docs/deployment/).

[configuration file]: https://gohugo.io/getting-started/configuration/#configuration-file