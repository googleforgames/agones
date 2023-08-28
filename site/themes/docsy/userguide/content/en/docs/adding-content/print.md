---
title: "Print Support"
linkTitle: "Print Support"
weight:  7
description: >
     Making it easier to print entire sections of documentation.
---

Individual documentation pages print well from most browsers as the layouts have been styled to omit navigational chrome from the printed output.

On some sites, it can be useful to enable a "print entire section" feature (as seen in this user guide).  Selecting this option renders the entire current top-level section (such as Content and Customization for this page) with all of its child pages and sections in a format suited to printing, complete with a table of contents for the section.

To enable this feature, add the "print" output format in your site's `hugo.toml`/`hugo.yaml`/`hugo.json` file for the "section" type:

{{< tabpane persistLang=false >}}
{{< tab header="Configuration file:" disabled=true />}}
{{< tab header="hugo.toml" lang="toml" >}}
[outputs]
section = [ "HTML", "RSS", "print" ]
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
outputs:
  section:
    - HTML
    - RSS
    - print
{{< /tab >}}
{{< tab header="hugo.json" lang="json" >}}
{
  "outputs": {
    "section": [
      "HTML",
      "RSS",
      "print"
    ]
  }
}
{{< /tab >}}
{{< /tabpane >}}

The site should then show a "Print entire section" link in the right hand navigation.

## Further Customization

### Disabling the ToC

To disable showing the the table of contents in the printable view, set the `disable_toc` param to `true`, either in the page front matter, or in `hugo.toml`/`hugo.yaml`/`hugo.json`:

{{< tabpane persistLang=false >}}
{{< tab header="Front matter:" disabled=true />}}
{{< tab header="toml" lang="toml" >}}
+++
…
disable_toc = true
…
+++
{{< /tab >}}
{{< tab header="yaml" lang="yaml" >}}
---
…
disable_toc: true
…
---
{{< /tab >}}
{{< tab header="json" lang="json" >}}
{
  …,
  "disable_toc": true,
  …
}
{{< /tab >}}
{{< tab header="or config file:" disabled=true />}}
{{< tab header="hugo.toml" lang="toml" >}}
[params.print]
disable_toc = true
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
params:
  print:
    disable_toc: true
{{< /tab >}}
{{< tab header="hugo.json" lang="json" >}}
{
  "params": {
    "print": {
      "disable_toc": true
    }
  }
}
{{< /tab >}}
{{< /tabpane >}}

## Layout hooks

A number of layout partials and hooks are defined that can be used to customize the printed format.  These can be found in `layouts/partials/print`.

Hooks can be defined on a per-type basis.  For example, you may want to customize the layouts of heading for "blog" pages vs "docs". This can be achieved by creating `layouts/partials/print/page-heading-<type>.html` - eg. `page-heading-blog.html`.  It defaults to using the page title and description as a heading.

Similarly, the formatting for each page can be customized by creating `layouts/partials/print/content-<type>.html`.




