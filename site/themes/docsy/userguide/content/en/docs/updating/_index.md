---
title: "Update Docsy"
linkTitle: "Update Docsy"
weight: 8
description: >
 Keeping the Docsy theme up to date.
---

We hope to continue to make improvements to the theme [along with the Docsy community](/docs/contribution-guidelines/).
If you have cloned the example site (or are otherwise using the theme as a Hugo Module or Git submodule), you can easily update the Docsy theme in your site yourself. If you have cloned the theme itself into your own project you can also update, though you may need to resolve merge conflicts.

Updating Docsy means that your site will build using the latest version of Docsy at `HEAD` and include 
all the new commits or changes that have been merged since the point in time that you initially added the Docsy 
submodule, or last updated. Updating won't affect any modifications that you made in your own project to 
[override the Docsy look and feel](/docs/adding-content/lookandfeel/), as your overrides 
don't modify the theme itself. For details about what has changed in the theme since your last update, see the list of 
[Docsy commits](https://github.com/google/docsy/commits/main).

If you have been using the theme as a Git submodule, you can also update your site to use [Docsy as a Hugo Module](/docs/get-started/docsy-as-module/). This is the latest and simplest way to pull in a Hugo theme from its repository. If you're not ready to migrate to Hugo Modules yet, don't worry, your site will still work and you can continue to update your submodule as before.
