# Docsy

Docsy is a [Hugo](https://gohugo.io) theme for technical documentation sets,
providing simple navigation, site structure, and more.

This is not an officially supported Google product. This project is actively
being maintained.

## Prerequisites

The following are basic prerequisites for using Docsy in your site:

- Install a recent release of the Hugo "extended" version. If you install from
  the [Hugo release page](https://github.com/gohugoio/hugo/releases), make sure
  you download the `_extended` version which supports SCSS.

- Install `PostCSS` so that the site build can create the final CSS assets. You
  can install it locally by running the following commands from the root
  directory of your project:

  ```console
  $ npm install --save-dev autoprefixer
  $ npm install --save-dev postcss-cli
  ```
  
  Starting in [version 8 of `postcss-cli`](https://github.com/postcss/postcss-cli/blob/master/CHANGELOG.md),
  you must also separately install `postcss`:

  ```bash
  npm install -D postcss
  ```
  
Any additional prerequisites depend on the [installation option](https://www.docsy.dev/docs/get-started/#installation-options)
you choose. We recommend using Docsy as a Hugo module, which requires that
you have the `go` language installed in addition to Hugo and PostCSS.

For complete prerequisites and instructions, see our [Get started guides](https://www.docsy.dev/docs/get-started/).

## Example and usage

You can find an example project that uses Docsy in the [Docsy Example Project
repo](https://github.com/google/docsy-example).The Docsy Example Project is
hosted at [example.docsy.dev](https://example.docsy.dev). For
real-life examples of sites that use Docsy (and their source repos), see our
[Examples](https://www.docsy.dev/docs/examples/) page.

To use the Docsy theme for your own site:

- (Recommended) Use the [example
  project](https://github.com/google/docsy-example), which includes the Docsy
  theme as a Hugo module, as a template to create your project. You can customize
  this pre-configured basic site into your own Docsy themed site. [Learn
  more...](https://github.com/google/docsy-example)

- Add Docsy to your existing Hugo site. You can
  add Docsy as a Hugo module, as a Git submodule, or clone the Docsy theme into your
  project.

See the [Get started guides](https://www.docsy.dev/docs/get-started/)
for details about the various usage options.

## Documentation

Docsy has its own user guide (using Docsy, of course!) with lots more
information about using the theme. It is hosted by [Netlify][] at
[docsy.dev](https://docsy.dev). For deploy logs and more, see [Deploys][] from
the site's Netlify dashboard.

Alternatively you can use Hugo to generate and serve a local copy of the guide
(also useful for testing local theme changes), making sure you have installed
all the prerequisites listed above:

```console
$ git clone --depth 1 https://github.com/google/docsy.git
$ cd docsy/userguide/
$ npm install
$ npm run serve
```

## Contributing ![GitHub](https://img.shields.io/github/contributors/google/docsy)

Please read
[CONTRIBUTING.md](https://github.com/google/docsy/blob/main/CONTRIBUTING.md)
for details on our code of conduct, and the process for submitting pull requests
to us. See also the list of
[contributors](https://github.com/google/docsy/graphs/contributors) who
participated in this project.

## License ![GitHub](https://img.shields.io/github/license/google/docsy)

This project is licensed under the Apache License 2.0 - see the
[LICENSE.md](https://github.com/google/docsy/blob/main/LICENSE) file for
details

[Deploys]: https://app.netlify.com/sites/docsydocs/deploys
[Netlify]: https://netlify.com
