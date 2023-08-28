---
title: Before you begin
date: 2021-12-08T11:12:59+01:00
weight: 1
description: >
  Prerequisites for building a site with Docsy as a Hugo Module.
---

This page describes the prerequisites for building a site that uses Docsy as a Hugo Module.

## Install Hugo

You need a [recent **extended** version](https://github.com/gohugoio/hugo/releases) (we recommend version 0.73.0 or later) of [Hugo](https://gohugo.io/) to do local builds and previews of sites (like this one) that use Docsy. If you install from the release page, make sure to get the `extended` Hugo version, which supports [SCSS](https://sass-lang.com/documentation/file.SCSS_FOR_SASS_USERS.html); you may need to scroll down the list of releases to see it.

For comprehensive Hugo documentation, see [gohugo.io](https://gohugo.io).

### On Linux

Be careful using `sudo apt-get install hugo`, as it [doesn't get you the `extended` version for all Debian/Ubuntu versions](https://gohugo.io/getting-started/installing/#debian-and-ubuntu), and may not be up-to-date with the most recent Hugo version.

If you've already installed Hugo, check your version:

```bash
hugo version
```

If the result is `v0.73` or earlier, or if you don't see `Extended`, you'll need to install the latest version. You can see a complete list of Linux installation options in [Install Hugo](https://gohugo.io/getting-started/installing/#linux). The following shows you how to install Hugo from the release page:

1.  Go to the [Hugo releases](https://github.com/gohugoio/hugo/releases) page.
2.  In the most recent release, scroll down until you find a list of
    **Extended** versions.
3.  Download the latest extended version (`hugo_extended_0.1XX_Linux-64bit.tar.gz`).
4.  Create a new directory:

    ```bash
    mkdir hugo
    ```

5.  Extract the files you downloaded to `hugo`.

6.  Switch to your new directory:

    ```bash
    cd hugo
    ```

7.  Install Hugo:

    ```bash
    sudo install hugo /usr/bin
    ```

### On macOS

Install Hugo using [Brew](https://gohugo.io/getting-started/installing/#homebrew-macos).

### As an `npm` module

You can install Hugo as an `npm` module using [`hugo-bin`](https://www.npmjs.com/package/hugo-bin). This adds `hugo-bin` to your `node_modules` folder and adds the dependency to your `package.json` file.  To install the extended version of Hugo:

```bash
npm install hugo-extended --save-dev
```

See the [`hugo-bin` documentation](https://www.npmjs.com/package/hugo-bin) for usage details.

## Install Go language

Hugo's commands for module management require that the Go programming language is installed on your system. Check whether `go` is already installed:

```console
$ go version
go version go1.20.5
```

Ensure that you are using version 1.12 or higher.

If the `go` language is not installed on your system yet or if you need to upgrade, go to the [download area](https://go.dev/dl/) of the Go website, choose the installer for your system architecture and execute it. Afterwards, check for a successful installation.


## Install Git VCS client

Hugo's commands for module management require that the `git` client is installed on your system. Check whether `git` is already present in your system:

```console
$ git version
git version 2.41.0
```

If no `git` client is installed on your system yet, go to the [Git website](https://git-scm.com/), download the installer for your system architecture and execute it. Afterwards, check for a successful installation.

## Install PostCSS

To build or update your site's CSS resources, you also need [`PostCSS`](https://postcss.org/) to create the final assets. If you need to install it, you must have a recent version of [NodeJS](https://nodejs.org/en/) installed on your machine so you can use `npm`, the Node package manager. By default `npm` installs tools under the directory where you run [`npm install`](https://docs.npmjs.com/cli/v6/commands/npm-install#description):

```bash
npm install -D autoprefixer
npm install -D postcss-cli
```

Starting in [version 8 of `postcss-cli`](https://github.com/postcss/postcss-cli/blob/master/CHANGELOG.md), you must also separately install `postcss`:

```bash
npm install -D postcss
```

Note that versions of `PostCSS` later than 5.0.1 will not load `autoprefixer` if installed [globally](https://flaviocopes.com/npm-packages-local-global/), you must use a local install.


## Install/Upgrade Node.js

To ensure you can properly build your site beyond executing `hugo server`, you must have the [latest long term support (LTS) Version](https://nodejs.org/en/about/releases/) of Node.js. If you do not have the latest LTS version, you may see the one of following errors:

```
Error: Error building site: POSTCSS: failed to transform "scss/main.css" (text/css): Unexpected identifier
#OR
/home/user/repos/my-new-site/themes/docsy/node_modules/hugo-extended/postinstall.js:1
import install from "./lib/install.js";
       ^^^^^^^

SyntaxError: Unexpected identifier
    at Module._compile (internal/modules/cjs/loader.js:723:23)
    at Object.Module._extensions..js (internal/modules/cjs/loader.js:789:10)
    at Module.load (internal/modules/cjs/loader.js:653:32)
    at tryModuleLoad (internal/modules/cjs/loader.js:593:12)
    at Function.Module._load (internal/modules/cjs/loader.js:585:3)
    at Function.Module.runMain (internal/modules/cjs/loader.js:831:12)
    at startup (internal/bootstrap/node.js:283:19)
    at bootstrapNodeJSCore (internal/bootstrap/node.js:623:3)

```

You can check your current Node.js version by running `node -v`.  If you need to install a new version, see the following instructions:

* [Debian and Ubuntu based distributions](https://github.com/nodesource/distributions/blob/master/README.md#installation-instructions)

   tl;dr:

   ```bash
   # Using Ubuntu
   curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
   sudo apt-get install -y nodejs

   # Using Debian, as root
   curl -fsSL https://deb.nodesource.com/setup_18.x | bash -
   apt-get install -y nodejs
   ```

* [Enterprise Linux based distributions](https://github.com/nodesource/distributions/blob/master/README.md#installation-instructions-1)

   tl;dr:

   ```bash
   # As root
   curl -fsSL https://rpm.nodesource.com/setup_18.x | bash -

   # No root privileges
   curl -fsSL https://rpm.nodesource.com/setup_18.x | sudo bash -
   ```


## What's next?

With all prerequisites installed, choose how to start off with your new Hugo site

* [Start with a prepopulated site (for beginners)](/docs/get-started/docsy-as-module/example-site-as-template/)
* [Start site from scratch (for experts)](/docs/get-started/docsy-as-module/start-from-scratch/)
