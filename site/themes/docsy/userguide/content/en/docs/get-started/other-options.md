---
title: Other setup options
description: Create a new Docsy site with Docsy using Git or NPM
date: 2021-12-08T09:22:27+01:00
spelling: cSpell:ignore docsy gohugo hugo myproject
weight: 2
---

If you don't want to use
[Docsy as a Hugo Module](/docs/get-started/docsy-as-module/) (for example if you
do not want to install Go) but still don't want to copy the theme files into
your own repo, you can **use Docsy as a
[Git submodule](https://git-scm.com/book/en/v2/Git-Tools-Submodules)**. Using
submodules also lets Hugo use the theme files from Docsy repo, though is more
complicated to maintain than the Hugo Modules approach. This is the approach
used in older versions of the Docsy example site, and is still supported. If you
are using Docsy as a submodule but would like to migrate to Hugo Modules, see
our [migration guide]().

Alternatively if you don’t want Hugo to have to get the theme files from an
external repo (for example, if you want to customize and maintain your own copy
of the theme directly, or your deployment choice requires you to include a copy
of the theme in your repository), you can **clone the files directly into your
site source**.

Finally, you can **install
[Docsy as an NPM package](#option-3-docsy-as-an-npm-package)**.

This guide provides instructions for all of these options, along with common
prerequisites.

## Prerequisites

### Install Hugo

You need a
[recent **extended** version](https://github.com/gohugoio/hugo/releases) (we
recommend version 0.73.0 or later) of [Hugo](https://gohugo.io/) to do local
builds and previews of sites (like this one) that use Docsy. If you install from
the release page, make sure to get the `extended` Hugo version, which supports
[SCSS](https://sass-lang.com/documentation/file.SCSS_FOR_SASS_USERS.html); you
may need to scroll down the list of releases to see it.

For comprehensive Hugo documentation, see [gohugo.io](https://gohugo.io/).

#### On Linux

Be careful using `sudo apt-get install hugo`, as it
[doesn't get you the `extended` version for all Debian/Ubuntu versions](https://gohugo.io/getting-started/installing/#debian-and-ubuntu),
and may not be up-to-date with the most recent Hugo version.

If you've already installed Hugo, check your version:

```
hugo version
```

If the result is `v0.73` or earlier, or if you don't see `Extended`, you'll need
to install the latest version. You can see a complete list of Linux installation
options in [Install Hugo](https://gohugo.io/getting-started/installing/#linux).
The following shows you how to install Hugo from the release page:

1.  Go to the [Hugo releases](https://github.com/gohugoio/hugo/releases) page.
2.  In the most recent release, scroll down until you find a list of
    **Extended** versions.
3.  Download the latest extended version
    (`hugo_extended_0.9X_Linux-64bit.tar.gz`).
4.  Create a new directory:

        mkdir hugo

5.  Extract the files you downloaded to `hugo`.

6.  Switch to your new directory:

        cd hugo

7.  Install Hugo:

        sudo install hugo /usr/bin

#### On macOS

Install Hugo using
[Brew](https://gohugo.io/getting-started/installing/#homebrew-macos).

#### As an NPM module

You can install Hugo as an NPM module using
[hugo-extended](https://www.npmjs.com/package/hugo-extended). To install the
extended version of Hugo:

```
npm install hugo-extended --save-dev
```

### Node: Get the latest LTS release

If you have Node installed already, check your version of Node. For example:

```sh
node -v
```

Install or upgrade your version of Node to the **active [LTS release][]**. We
recommend using **[nvm][]** to manage your Node installation (Linux command
shown):

```sh
nvm install --lts
```

### Install PostCSS

To build or update your site's CSS resources, you'll also need
[PostCSS](https://postcss.org/). Install it using the Node package manager,
`npm`.

{{% alert title="IMPORTANT: Check your Node version" color="warning" %}}

The PostCSS package installed by some older versions of Node is incompatible
with Docsy. Check your version of Node against the **active [LTS release][]**
and upgrade, if necessary. For details, see [Node: Get the latest LTS
release][latest-lts].

[lts release]: https://nodejs.org/en/about/releases/
[latest-lts]: #node-get-the-latest-lts-release

{{% /alert %}}

From your project root, run this command:

```
npm install --save-dev autoprefixer postcss-cli postcss
```

## Option 1: Docsy as a Git submodule

### For a new site

To create a **new site** and add the Docsy theme as a Git submodule, run the
following commands:

1.  Create the site:

    ```shell
    hugo new site myproject
    cd myproject
    git init
    ```

2.  Install postCSS as [instructed earlier](#install-postcss).

3.  Follow the instructions below for an existing site.

### For an existing site

To add the Docsy theme to an **existing site**, run the following commands from
your project's root directory:

1.  Install Docsy as a Git submodule:

    ```sh
    git submodule add https://github.com/google/docsy.git themes/docsy
    cd themes/docsy
    git checkout v{{% param version %}}
    ```

    To work from the development version of Docsy (not recommended),
    run the following command instead:

    ```sh
    git submodule add --depth 1 https://github.com/google/docsy.git themes/docsy
    ```

2.  Add Docsy as a theme, for example:

    ```sh
    echo 'theme = "docsy"' >> hugo.toml
    ```

    {{% alert title="Tip" %}}
In Hugo 0.110.0 the default config base filename was changed to `hugo.toml`.
If you are using hugo 0.110 or above, consider renaming your `config.toml` to `hugo.toml`!
    {{% /alert %}}

3.  Get Docsy dependencies:

    ```sh
    (cd themes/docsy && npm install)
    ```

4.  (Optional but recommended) To avoid having to repeat the previous step every
    time you update Docsy, consider adding [NPM scripts][] like the following to
    your project's `package.json` file:

    ```json
    {
      "...": "...",
      "scripts": {
        "get:submodule": "git submodule update --init --depth 1",
        "_prepare:docsy": "cd themes/docsy && npm install",
        "prepare": "npm run get:submodule && npm run _prepare:docsy",
        "...": "..."
      },
      "...": "..."
    }
    ```

    Every time you run `npm install` from your project root, the `prepare`
    script will fetch the latest version of Docsy and its dependencies.

From this point on, build and serve your site using the usual Hugo commands, for
example:

```sh
hugo serve
```

## Option 2: Clone the Docsy theme

If you don't want to use a submodules (for example, if you want to customize and
maintain your own copy of the theme directly, or your deployment choice requires
you to include a copy of the theme in your repository), you can clone the theme
into your project's `themes` subdirectory.

To clone Docsy at v{{% param version %}} into your project's `theme` folder, run
the following commands from your project's root directory:

```sh
cd themes
git clone -b v{{% param version %}} https://github.com/google/docsy
cd docsy
npm install
```

To work from the development version of Docsy (not recommended unless, for
example, you plan to upstream changes to Docsy), omit the `-b v{{% param version
%}}` argument from the clone command above.

Then consider setting up an NPM [prepare][] script, as documented in Option 1.

For more information, see
[Theme Components](https://gohugo.io/hugo-modules/theme-components/) on the
[Hugo](https://gohugo.io) site.

## Option 3: Docsy as an NPM package

You can use Docsy as an NPM module as follows:

1.  Create your site and specify Docsy as the site theme:

    ```sh
    hugo new site myproject
    cd myproject
    echo 'theme = "docsy"' >> hugo.toml
    ```

2.  Install Docsy, and postCSS (as [instructed earlier](#install-postcss)):

    ```console
    npm install --save-dev google/docsy#semver:{{% param version %}} autoprefixer postcss-cli postcss
    ```

3.  Build or serve your new site using the usual Hugo commands, specifying the
    path to the Docsy theme files. For example, build your site as follows:

    ```console
    $ hugo --themesDir node_modules
    Start building sites …
    ...
    Total in 1890 ms
    ```

    You can drop the `--themesDir ...` flag by adding the themes directory to
    your site's configuration file:

    ```sh
    echo 'themesDir = "node_modules"' >> hugo.toml
    ```

As an alternative to specifying a `themesDir`, on some platforms, you can
instead create a symbolic link to the Docsy theme directory as follows (Linux
commands shown, executed from the site root folder):

```sh
mkdir -p themes
pushd themes
ln -s ../node_modules/docsy
popd
```

## Preview your site

To preview your site locally:

```sh
cd myproject
hugo server
```

By default, your site will be available at <http://localhost:1313>.
[See the known issues on MacOS](/docs/get-started/known_issues/#macos).

You may get Hugo errors for missing parameters and values when you try to build
your site. This is usually because you’re missing default values for some
configuration settings that Docsy uses - once you add them your site should
build correctly. You can find out how to add configuration in
[Basic site configuration](/docs/get-started/basic-configuration/) - we
recommend copying the example site configuration even if you’re creating a site
from scratch as it provides defaults for many required configuration parameters.

## What's next?

- Add some [basic site configuration](/docs/get-started/basic-configuration/)
- [Add content and customize your site](/docs/adding-content/)
- Get some ideas from our
  [Example Site](https://github.com/google/docsy-example) and other
  [Examples](/docs/examples/).
- [Publish your site](/docs/deployment/).

[lts release]: https://nodejs.org/en/about/releases/
[nvm]:
  https://github.com/nvm-sh/nvm/blob/master/README.md#installing-and-updating
[npm scripts]: https://docs.npmjs.com/cli/v8/using-npm/scripts
[prepare]:
  https://docs.npmjs.com/cli/v8/using-npm/scripts#prepare-and-prepublish
