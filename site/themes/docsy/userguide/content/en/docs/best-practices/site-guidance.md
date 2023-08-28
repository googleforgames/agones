---
title: "Hugo Content Tips"
linkTitle: "Hugo Content Tips"
weight: 9
description: >
  Tips for authoring content for your Docsy-themed Hugo site.
---

Docsy is a theme for the [Hugo](https://gohugo.io/) static site generator.
If you're not already familiar with Hugo this page provides some useful tips and
potential gotchas for adding and editing content for your site. Feel free to add your own!

## Linking

By default, regular relative URLs in links are left unchanged by Hugo (they're still relative links in your site's generated HTML), hence some hardcoded relative links like `[relative cross-link](../../peer-folder/sub-file.md)` might behave unexpectedly compared to how they work on your local file system. You may find it helpful to use some of Hugo's built-in [link shortcodes](https://gohugo.io/content-management/cross-references/#use-ref-and-relref) to avoid broken links in your generated site. For example a `{{</* ref "filename.md" */>}}` link in Hugo will actually
find and automatically link to your file named `filename.md`.

Note, however, that `ref` and `relref` links don't work with `_index` or `index` files (for example, this site's [content landing page](/docs/adding-content/)): you'll need to use regular Markdown links to section landing or other index pages. Specify these links relative to the site's root URL, for example: `/docs/adding-content/`.

[Learn more about linking](/docs/adding-content/content/#working-with-links).

