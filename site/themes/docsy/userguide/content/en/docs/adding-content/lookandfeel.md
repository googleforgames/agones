---
title: Look and Feel
date: 2017-01-05
weight: 2
description: Customize colors, fonts, code highlighting, and more for your site.
spelling: cSpell:ignore wordmark docsy
---

By default, a site using Docsy has the theme's default fonts, colors, and general look and feel. However, if you want your own color scheme (and you probably will!) you can very easily override the theme defaults with your own project-specific values - Hugo will look in your project files first when looking for information to build your site. And because Docsy uses Bootstrap 4 and SCSS for styling, you can override just single values (such as project colors and fonts) in its special SCSS project variables file, or do more serious customization by creating your own styles.

Docsy also provides options for styling your code blocks, using either Chroma or Prism for highlighting.

## Project style files

To customize your project's look and feel, create your own version of either or both of the following
Docsy placeholder files (note the **`_project.scss`** suffixes):

- [`assets/scss/`**`_variables_project.scss`**][_variables_project] is where you add project-specific definitions of theme variables such as [site colors](#site-colors), as well as any additional Bootstrap variable values you want to set. You can find a list of Docsy's theme variables and their default values in [<code>assets/scss/<strong>_variables.scss</strong></code>][_variables].  For information about other Bootstrap 4 variables, see [Variable defaults][] and Bootstrap's [v4-dev/scss/_variables.scss][] file.
- [`assets/scss/`**`_styles_project.scss`**][_styles_project] is where you can add your own custom SCSS styles, including overriding any of the styles in Docsy's theme SCSS files.

[_styles_project]: https://github.com/google/docsy/blob/main/assets/scss/_styles_project.scss
[_variables_project]: https://github.com/google/docsy/blob/main/assets/scss/_variables_project.scss
[_variables]: https://github.com/google/docsy/blob/main/assets/scss/_variables.scss
[v4-dev/scss/_variables.scss]: https://github.com/twbs/bootstrap/blob/v4-dev/scss/_variables.scss
[Variable defaults]: https://getbootstrap.com/docs/4.1/getting-started/theming/#variable-defaults

{{% alert title="Tip" %}}
PostCSS (autoprefixing of CSS browser-prefixes) is not enabled when running in server mode (it is a little slow), so Chrome is the recommended choice for development.
{{% /alert %}}

## Site colors

To customize your site's colors, add SCSS variable overrides to
`assets/scss/_variables_project.scss`. For example, you can set the primary and
secondary site colors as follows:

```scss
$primary: #390040;
$secondary: #A23B72;
```

The theme has features such as gradient backgrounds (`$enable-gradients`) and
shadows (`$enable-shadows`) enabled by default. These can also be toggled in
your project variables file by setting the variables to `false`.

## Fonts

The theme uses [Open Sans](https://fonts.google.com/specimen/Open+Sans) as its primary font. To disable Google Fonts and use a system font, set this SCSS variable in `assets/scss/_variables_project.scss`:

```scss
$td-enable-google-fonts: false;
```

To configure another Google Font:

```scss
$google_font_name: "Open Sans";
$google_font_family: "Open+Sans:300,300i,400,400i,700,700i";
```

Note that if you decide to go with a font with different weights (in the built-in configuration this is `300` (light), `400` (medium) and `700` (bold)), you also need to adjust the weight related variables, i.e. variables starting with `$font-weight-`.

## CSS utilities

For documentation of available CSS utility classes, see the [Bootstrap Documentation](https://getbootstrap.com/). This theme adds very little on its own in this area. However, we have added some color state CSS classes that can be useful in a dynamic context:

* `.-bg-<color>`
* `.-text-<color>`

You can use these classes, for example, to style your text in an appropriate color when you don't know if the `primary` color is dark or light, to ensure proper color contrast. They are also useful when you receive the color code as a [shortcode](/docs/adding-content/shortcodes/) parameter.

The value of `<color>` can be any of the color names, `primary`, `white`, `dark`, `warning`, `light`, `success`, `300`, `blue`, `orange` etc.

When you use `.-bg-<color>`, the text colors will be adjusted to get proper contrast:

```html
<div class="-bg-primary p-3 display-4">Background: Primary</div>
<div class="-bg-200 p-3 display-4">Background: Gray 200</div>
```

<div class="-bg-primary p-3 display-4 w-75">Background: Primary</div>
<div class="-bg-200 p-3 display-4 mb-5 w-50 w-75">Background: Gray 200</div>

`.-text-<color>` sets the text color only:

```html
<div class="-text-blue pt-3 display-4">Text: Blue</div>
```

<div class="-text-blue pt-3 display-4">Text: Blue</div>

## Code highlighting with Chroma

With Hugo version 0.60 and higher, you can choose from a range of code block highlight and colour styles using [Chroma](https://github.com/alecthomas/chroma) that are applied to your fenced code blocks by default. If you copied a recent `hugo.toml` your site uses Tango (like this site), otherwise the Hugo default is Monokai. You can switch to any of the [available Chroma styles](https://xyproto.github.io/splash/docs/all.html) (including our Docsy default Tango) using your `hugo.toml`/`hugo.yaml`/`hugo.json`:

{{< tabpane persistLang=false >}}
{{< tab header="Configuration file:" disabled=true />}}
{{< tab header="hugo.toml" lang="toml" >}}
[markup]
  [markup.goldmark]
    [markup.goldmark.renderer]
      unsafe = true
  [markup.highlight]
      # See a complete list of available styles at https://xyproto.github.io/splash/docs/all.html
      style = "tango"
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
markup:
  goldmark:
    renderer:
      unsafe: true
  highlight:
    style: tango
{{< /tab >}}
{{< tab header="hugo.json" lang="json" >}}
{
  "markup": {
    "goldmark": {
      "renderer": {
        "unsafe": true
      }
    },
    "highlight": {
      "style": "tango"
    }
  }
}
{{< /tab >}}
{{< /tabpane >}}

By default code highlighting styles are not applied to code blocks without a specified language, instead you get Docsy's default style of grey with black text. If you would like the code highlighting style to apply to all code blocks, even without a language, uncomment or add the following line under `[markup.highlight]` in your `hugo.toml`/`hugo.yaml`/`hugo.json`.

{{< tabpane persistLang=false >}}
{{< tab header="Configuration file:" disabled=true />}}
{{< tab header="hugo.toml" lang="toml" >}}
# Uncomment if you want your chosen highlight style used for code blocks without a specified language
guessSyntax = true
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
guessSyntax: true
{{< /tab >}}
{{< tab header="hugo.json" lang="json" >}}
"guessSyntax": true
{{< /tab >}}
{{< /tabpane >}}

If you are using a Docsy version later than `0.6.0`, the code blocks show a
"Copy to clipboard" icon in the top right-hand corner. To disable this
functionality set `disable_click2copy_chroma` to `true` in your configuration
file:

You can find out more about code highlighting in Hugo with Chroma in [Syntax Highlighting](https://gohugo.io/content-management/syntax-highlighting/).

## Code highlighting with Prism

Optionally, you can enable Prism syntax highlighting in your `hugo.toml`/`hugo.yaml`/`hugo.json`:

{{< tabpane persistLang=false >}}
{{< tab header="Configuration file:" disabled=true />}}
{{< tab header="hugo.toml" lang="toml" >}}
# Enable syntax highlighting and copy buttons on code blocks with Prism
prism_syntax_highlighting = true
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
prism_syntax_highlighting: true
{{< /tab >}}
{{< tab header="hugo.json" lang="json" >}}
"prism_syntax_highlighting": true
{{< /tab >}}
{{< /tabpane >}}

When this option is enabled your site uses [Prism](https://prismjs.com/index.html) instead of Chroma for code block highlighting.

Prism is a popular open source syntax highlighter which supports over 200 [languages](https://prismjs.com/index.html#supported-languages) and various [plugins](https://prismjs.com/index.html#plugins).

Docsy includes JavaScript and CSS files for a basic Prism configuration, which supports:

* Code blocks styled with the Prism `Default` theme
* Copy to clipboard buttons on code blocks
* Syntax highlighting for a number of common languages, as specified in the following Prism download link, [Customize your download][prismjs-download+].

[prismjs-download+]: https://prismjs.com/download.html#themes=prism&languages=markup+css+clike+javascript+bash+c+csharp+cpp+go+java+markdown+python+scss+sql+toml+yaml&plugins=toolbar+copy-to-clipboard

### Code blocks with no language

By default, Prism code highlighting styles are not applied to code blocks without a specified language, instead you get Docsy's default style of grey with black text. To apply Prism styling to code blocks with no language or a language not supported by Prism, specify `none` as the language after your triple backticks.

### Extending Prism for additional languages or plugins

If the included Prism configuration is not sufficient for your requirements, and you want to use additional languages or plugins you can replace the included files with your own.

1. Download your own Prism JS and CSS files from <https://prismjs.com/download.html>
2. Replace the included Prism JS and CSS with the files you downloaded:
    * Copy the Javascript file to `static/js/prism.js`
    * Copy the CSS file to `static/css/prism.css`


## Navbar

For pages containing a [blocks/cover][] shortcode, like most homepages, the
navbar is translucent as long as the hero image hasn't scrolled up past the
navbar. For an example, see the [About Docsy][] page. This initial translucent
setting ensures that the hero image is maximally visible.

After the hero image has scrolled past the navbar, the navbar's (opaque)
background color is set -- usually to the site's [primary color][].

The text of navbar entries can be difficult to read with some hero images. In
these cases, you can disable navbar translucency by setting the
`params.ui.navbar_translucent_over_cover_disable` option to `true` in your
site's [configuration file][].

[About Docsy]: https://www.docsy.dev/about/
[blocks/cover]: /docs/adding-content/shortcodes/#blockscover
[configuration file]: https://gohugo.io/getting-started/configuration/#configuration-file
[primary color]: #site-colors

### Styling your project logo and name

The default Docsy navbar (`.td-navbar`) displays your site identity, consisting
of the following:

1.  [Your logo][], which is included in the navbar as an inline SVG, styled by
    `.td-navbar .navbar-brand svg`. For the style details, see [_nav.scss][].

    To ensure your logo displays correctly, you may want to resize it and ensure
    that it doesn't have height and width attributes so that its size is fully
    responsive. [Override the default styling][project-styles] of `.td-navbar
    .navbar-brand svg` or (equivalently) `.td-navbar .navbar-brand__logo` as
    needed.
2.  Your project name, which is the site `title`. If you don't want your project
    name to appear (for example, because your logo is or contains a
    [wordmark][]), then add the following custom styling to your [project's
    styles][project-styles]:

    ```css
    .td-navbar .navbar-brand__name {
      display: none;
    }
    ```

[_nav.scss]: https://github.com/google/docsy/blob/main/assets/scss/_nav.scss
[project-styles]: /docs/adding-content/lookandfeel/#project-style-files
[wordmark]: https://en.wikipedia.org/wiki/Wordmark
[your logo]: /docs/adding-content/iconsimages/#add-your-logo

## Tables

Docsy applies the following styles to all tables, through the class `.td-table`:

- [Bootstrap table][] styles:
  - `.table`
  - `.table-striped`
  - `.table-responsive`
- `display: block`, which is necessary for tables to be responsive.

This styling configuration gives you responsive tables using Markdown only,
without the need to wrap the table in a `<div>`. It does, however, mean that all your tables have `display`
set to `block`. If you don't want this, you can create your own custom styles for tables.

{{% alert title="Note" %}}

Our table styling goes against the [Bootstrap recommendation to _wrap_
tables][wrap-tables] with `.table-responsive` &mdash; however, we think letting
users create responsive tables with just Markdown table syntax is more
convenient.

[wrap-tables]: https://getbootstrap.com/docs/5.3/content/tables/#responsive-tables

{{% /alert %}}

To render a table without Docsy styling, apply the `.td-initial` class to the
table. From the resulting `<table>` style base, it is easier to apply your own
custom styles (rather than trying to undo Docsy table styling), as is
illustrated in the following example:

```markdown
| Shape    | Number of sides |
| -------- | --------------- |
| Triangle | 3               |
| Square   | 4               |
{.td-initial .my-dark-table-style}
```

The example above uses [Markdown attribute][] syntax, and might render like this:

| Shape    | Number of sides |
| -------- | --------------- |
| Triangle | 3               |
| Square   | 4               |
{.td-initial .table .table-dark}

[Bootstrap table]: https://getbootstrap.com/docs/5.3/content/tables/
[Markdown attribute]: https://discourse.gohugo.io/t/markdown-attributes/41783

## Customizing templates

### Add code to head or before body end

If you need to add some code (CSS import, cookie consent, or similar) to the `head` section on every page, add the `head-end.html` partial to your project:

```
layouts/partials/hooks/head-end.html
```

And add the code you need in that file. Your partial code is automatically included just before the end of the theme partial [`head.html`](https://github.com/google/docsy/blob/main/layouts/partials/head.html). The theme version of [`head-end.html`](https://github.com/google/docsy/blob/main/layouts/partials/hooks/head-end.html) is empty.


Similarly, if you want to add some code right before the `body` end, create your own version of the following file:

```
layouts/partials/hooks/body-end.html
```

Any code in this file is included automatically at the end of the theme partial [`scripts.html`](https://github.com/google/docsy/blob/main/layouts/partials/head.html).

Both `head.html` and `scripts.html` are then used to build Docsy's [base page layout](https://github.com/google/docsy/blob/main/layouts/_default/baseof.html), which is used by all the other page templates:

```html
<!doctype html>
<html lang="{{ .Site.Language.Lang }}" class="no-js">
  <head>
    {{ partial "head.html" . }}
  </head>
  <body class="td-{{ .Kind }}">
    <header>
      {{ partial "navbar.html" . }}
    </header>
    <div class="container-fluid td-default td-outer">
      <main role="main" class="td-main">
        {{ block "main" . }}{{ end }}
      </main>
      {{ partial "footer.html" . }}
    </div>
    {{ partialCached "scripts.html" . }}
  </body>
</html>
```

