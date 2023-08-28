---
title: "Use Docsy as a Hugo Module"
linkTitle: "Use Docsy as a Hugo Module"
weight: 1
date: 2021-12-08T10:33:16+01:00
description: >
  Learn how to get started with Docsy by using the theme as a Hugo Module.
---

[Hugo modules](https://gohugo.io/hugo-modules/) are the simplest and latest way to use Hugo themes like Docsy when building a website. Hugo uses the modules mechanism to pull in the theme files from the main Docsy repo at your chosen revision, and it’s easy to keep the theme up to date in your site. Our example site uses Docsy as a Hugo module.

To find out about other setup approaches, see our [Get started](/docs/get-started/) overview. If you want to migrate an existing Docsy site to use Hugo Modules, see our [migration guide](/docs/updating/convert-site-to-module/).

## Setup options with Hugo Modules

To use Docsy as a Hugo Module, you have a couple of options:

*   **Copy and edit the source for the [Docsy example site](https://github.com/google/docsy-example).** This approach gives you a skeleton structure for your site, with top-level and documentation sections and templates that you can modify as necessary. The example site uses Docsy as a Hugo Module.
*   **Build your own site using the Docsy theme.** Specify the [Docsy theme](https://github.com/google/docsy) like any other [Hugo theme](https://gohugo.io/themes/) when creating or updating your site. With this option, you'll get Docsy look and feel, navigation, and other features, but you'll need to specify your own site structure. 

If you're a beginner, we recommend that you get started by copying our example site. If you're already familiar with Hugo or want a very different site structure, you can follow our guide to start a site from scratch, which gives you maximum flexibility at the cost of higher implementation effort. In both cases you need to follow our prerequisites guide to make sure that you have installed Hugo and all necessary dependencies.
