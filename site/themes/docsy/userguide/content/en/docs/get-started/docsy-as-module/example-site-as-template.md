---
title: "Create a new site: start with a prepopulated site"
linkTitle: "Start with a prepopulated site"
date: 2021-12-08T09:21:54+01:00
weight: 2
description: >
  Create a new Hugo site by using a clone of the Docsy example site as your starting point.
---

The simplest way to create a new Docsy site is to use the source of the [Docsy example site](https://github.com/google/docsy-example) as starting point. This approach gives you a skeleton structure for your site, with top-level and documentation sections and templates that you can modify as necessary. The example site automatically pulls in the Docsy theme as a [Hugo Module](https://gohugo.io/hugo-modules/), so it's easy to [keep up to date](/docs/updating/updating-hugo-module/).

If you prefer to create a site from scratch, follow the instructions in Start a site from scratch.

## TL;DR: Setup for the impatient expert

At your Unix shell or Windows command line, run the following command:

```bash
git clone --depth 1 --branch v{{% param "version" %}} https://github.com/google/docsy-example.git my-new-site
cd  my-new-site
hugo server
```

You now can preview your new site in your browser at [http://localhost:1313](http://localhost:1313/).

## Detailed Setup instructions

### Clone the Docsy example site

The [Example Site](https://example.docsy.dev) gives you a good starting point for building your docs site and is
pre-configured to automatically pull in the Docsy theme as a Hugo Module.
There are two different routes to get a local clone of the example site:

* If you want to create a local copy only, choose option 1.
* If you have a GitHub account and want to create a GitHub repo for your site go for option 2.

#### Option 1: Using the command line (local copy only)

If you want  to use a remote repository other than GitHub (such as [GitLab](https://gitlab.com), [BitBucket](https://bitbucket.org/), [AWS CodeCommit](https://aws.amazon.com/codecommit/), [Gitea](https://gitea.io/)) or if you don't want a remote repo at all, simply make a local working copy of the example site directly using `git clone`. As last parameter, give your chosen local repo name (here: `my-new-site`):

```bash
git clone --depth 1 --branch v{{% param "version" %}} https://github.com/google/docsy-example.git my-new-site
```

#### Option 2: Using the GitHub UI (local copy + associated GitHub repo)

As the Docsy example site repo is a [template repository](https://github.blog/2019-06-06-generate-new-repositories-with-repository-templates/), creating your own remote GitHub clone of this Docsy example site repo is quite easy:

1. Go to the repo of the [Docsy example site](https://github.com/google/docsy-example).

1. Use the dropdown for switching branches/tags to change to the latest released tag `v{{% param "version" %}}`. 

1. Click the button **Use this template** and select the option `Create a new repository` from the dropdown.

1. Chose a name for your new repository (e.g. `my-new-site`) and type it in the **Repository name** field. You can also add an optional **Description**.

1. Click **Create repository from template** to create your new repository. Congratulations, you just created your remote Github clone which now serves as starting point for your own site!

1. Make a local copy of your newly created GitHub repository by using `git clone`, giving your repo's web URL as last parameter.

    ```bash
    git clone https://github.com/me-at-github/my-new-site.git
    ```

{{% alert title="Note" color="primary" %}}
Depending on your environment you may need to tweak the [module top level settings](https://github.com/google/docsy-example/blob/f88fca475c28ffba3d72710a50450870230eb3a0/hugo.toml#L222-L227) inside your `hugo.toml` slightly, for example by adding a proxy to use when downloading remote modules.
You can find details of what these configuration settings do in the [Hugo modules documentation](https://gohugo.io/hugo-modules/configuration/#module-config-top-level).
{{% /alert %}}

Now you can make local edits and test your copied site locally with Hugo.

### Preview your site

To build and preview your site locally, switch to the root of your cloned project and use hugo's `server` command:

```bash
cd my-new-site
hugo server
```

Preview your site in your browser at: [http://localhost:1313](http://localhost:1313/).
Thanks to Hugo's live preview, you can immediately see the effect of changes that you are making to the source files of your local repo.
Use `Ctrl + c` to stop the Hugo server whenever you like.
[See the known issues on MacOS](/docs/get-started/known_issues/#macos).

## What's next?

* Add some [basic configuration](/docs/get-started/basic-configuration/)
* [Edit existing content and add more pages](/docs/adding-content/)
* [Customize your site](/docs/adding-content/lookandfeel/)
* [Publish your site](/docs/deployment/).
