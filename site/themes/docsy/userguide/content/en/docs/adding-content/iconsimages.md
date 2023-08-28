---
title: Logos and Images
date: 2017-01-05
weight: 6
description: Add and customize logos, icons, and images in your project.
---

## Add your logo

By default, Docsy shows a site logo at the start of the navbar, that is, at the
extreme left. Place your project's SVG logo in `assets/icons/logo.svg`. This
overrides the default Docsy logo in the theme.

If you don't want a logo to appear in the navbar, then set `navbar_logo` to
`false` in your project's config:

{{< tabpane persistLang=false >}}
{{< tab header="Configuration file:" disabled=true />}}
{{< tab header="hugo.toml" lang="toml" >}}
navbar_logo = false
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
navbar_logo: false
{{< /tab >}}
{{< tab header="hugo.json" lang="json" >}}
{
  "navbar_logo": false
}
{{< /tab >}}
{{< /tabpane >}}

For information about styling your logo, see [Styling your project logo and
name][].

[Styling your project logo and name]: /docs/adding-content/lookandfeel/#styling-your-project-logo-and-name

## Use icons

Docsy includes the free FontAwesome icons by default, including logos for sites like GitHub and Stack Overflow. You can view all available icons in the [FontAwesome documentation](https://fontawesome.com/icons/), including the FontAwesome version when the icon was added and whether it is available for free tier users. Check Docsy's [`package.json`](https://github.com/google/docsy/blob/main/package.json) and release notes for Docsy's currently included version of FontAwesome.

You can add FontAwesome icons to your [top-level menu](/docs/adding-content/navigation/#adding-icons-to-the-top-level-menu), [section menu](/docs/adding-content/navigation/#add-icons-to-the-section-menu), or anywhere in your text.

## Add your favicons

The easiest way to do this is to create a set of favicons via http://cthedot.de/icongen (which lets you create a huge range of icon sizes and options from a single image) and/or [https://favicon.io](https://favicon.io), and put them in your site project's `static/favicons` directory. This will override the default favicons from the theme.

Note that https://favicon.io  doesn't create as wide a range of sizes as Icongen but *does* let you quickly create favicons from text: if you want to create text favicons you can use this site to generate them, then use Icongen to create more sizes (if necessary) from your generated `.png` file.

If you have special favicon requirements, you can create your own `layouts/partials/favicons.html` with your links.

## Add images

### Landing pages

Docsy's [`blocks/cover` shortcode](/docs/adding-content/shortcodes/#blockscover) make it easy to add large cover images to your landing pages. The shortcode looks for an image with the word "background" in the name inside the landing page's [Page Bundle](https://gohugo.io/content-management/page-bundles/) - so, for example, if you've copied the example site, the landing page image in `content/en/_index.html` is `content/en/featured-background.jpg`.

You specify the preferred display height of a cover block container (and hence its image) using the block's `height` parameter.  For a full viewport height, use `full`:

```html
{{</* blocks/cover title="Welcome to the Docsy Example Project!" image_anchor="top" height="full" */>}}
...
{{</* /blocks/cover */>}}
```

For a shorter image, as in the example site's About page, use one of `min`, `med`, `max` or `auto` (the actual height of the image):

```html
{{</* blocks/cover title="About the Docsy Example" image_anchor="bottom" height="min" */>}}
...
{{</* /blocks/cover */>}}
```

### Other pages

To add inline images to other pages, use the [`imgproc` shortcode](/docs/adding-content/shortcodes/#imgproc). Alternatively, if you prefer, just use regular Markdown or HTML images and add your image files to your project's `static` directory. You can find out more about using this directory in [Adding static content](/docs/adding-content/content/#adding-static-content).

## Images used on this site

Images used as background images in this site are in the [public domain](https://commons.wikimedia.org/wiki/User:Bep/gallery#Wed_Aug_01_16:16:51_CEST_2018) and can be used freely. The porridge image in the example site is by <a href="https://pixabay.com/users/iha31-560629/?utm_source=link-attribution&amp;utm_medium=referral&amp;utm_campaign=image&amp;utm_content=531209">iha31</a> from <a href="https://pixabay.com/?utm_source=link-attribution&amp;utm_medium=referral&amp;utm_campaign=image&amp;utm_content=531209">Pixabay</a>.

