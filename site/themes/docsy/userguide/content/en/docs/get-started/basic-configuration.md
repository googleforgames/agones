---
title: "Basic site configuration"
linkTitle: "Basic site configuration"
date: 2021-12-08T09:22:27+01:00
weight: 4
description: >
  Basic configuration for new Docsy sites.
---

Site-wide configuration details and parameters are defined in your project's [configuration file] (`hugo.toml` or `config.toml`). These include your chosen Hugo theme (Docsy, of course!), project name, community links, Google Analytics configuration, and Markdown parser parameters. See the examples with comments in [`hugo.toml` in the example project](https://github.com/google/docsy-example/blob/main/hugo.toml) for how to add this information. **We recommend copying this hugo.toml and editing it even if you’re just using the theme and not copying the entire Docsy example site**, as it includes default values for many parameters that you need to set for your site to build correctly.

You may want to remove or customize some defaults of the copied `hugo.toml` file straight away:

## Internationalization

The copied `hugo.toml` file defines content in English, Norwegian and Farsi. You can find out more about how Docsy supports multi-language content in [Multi-language support](/docs/language/).

If you don't intend to translate your site, you can remove the language switcher by removing the following lines from `hugo.toml`:

```toml
[languages.no]
title = "Docsy"
description = "Docsy er operativsystem for skyen"
languageName ="Norsk"
contentDir = "content/no"
time_format_default = "02.01.2006"
time_format_blog = "02.01.2006"

[languages.fa]
title = "اسناد گلدی"
description = "یک نمونه برای پوسته داکسی"
languageName ="فارسی"
contentDir = "content/fa"
time_format_default = "2006.01.02"
time_format_blog = "2006.01.02"
```

## Search

By default, the Docsy example site uses its own [Google Custom Search Engine](https://cse.google.com/cse/all). To disable this site search, delete or comment out the following lines:

```
# Google Custom Search Engine ID. Remove or comment out to disable search.
gcs_engine_id = "011737558837375720776:fsdu1nryfng"
```

To use your own Custom Search Engine, replace the value in the `gcs_engine_id` with the ID of your own search engine. Or [choose another search option](/docs/adding-content/navigation/#site-search-options).


## What's next?

* [Add content and customize your site](/docs/adding-content/)
* Get some ideas from our [Example Site](https://github.com/google/docsy-example) and other [Examples](/docs/examples/).
* [Publish your site](/docs/deployment/).

[configuration file]: https://gohugo.io/getting-started/configuration/#configuration-file
