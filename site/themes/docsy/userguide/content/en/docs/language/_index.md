---
title: "Multi-language Support"
linkTitle: "Multi-language Support"
weight: 7
description: >
  Support multiple languages in your site.
---

If you'd like to provide site content in multiple languages, the Docsy theme and Hugo make it easy to both add your translated content and for your users to navigate between language versions.

## Content and configuration

To add content in multiple languages, you first need to define the available languages in a `languages` section in your site configuration. Each language can have its own language-specific configuration. For example, the Docsy Example Site config specifies that it provides content in English and Norwegian, and that the language version visitors will see by default is English:

{{< tabpane persistLang=false >}}
{{< tab header="Configuration file:" disabled=true />}}
{{< tab header="hugo.toml" lang="toml" >}}
contentDir = "content/en"
defaultContentLanguage = "en"
defaultContentLanguageInSubdir = false
...
[languages]
[languages.en]
title = "Docsy"
description = "Docsy does docs"
languageName ="English"
# Weight used for sorting.
weight = 1
[languages.no]
title = "Docsy"
description = "Docsy er operativsystem for skyen"
languageName ="Norsk"
contentDir = "content/no"
time_format_default = "02.01.2006"
time_format_blog = "02.01.2006"
{{< /tab >}}
{{< tab header="hugo.yaml" lang="yaml" >}}
contentDir: content/en
defaultContentLanguage: en
defaultContentLanguageInSubdir: false
…
languages:
  en:
    title: Docsy
    description: Docsy does docs
    languageName: English
    weight: 1 # used for sorting
  'no':
    title: Docsy
    description: Docsy er operativsystem for skyen
    languageName: Norsk
    contentDir: content/no
    time_format_default: 02.01.2006
    time_format_blog: 02.01.2006
{{< /tab >}}
{{< tab header="hugo.json" lang="json" >}}
{
  "contentDir": "content/en",
  "defaultContentLanguage": "en",
  "defaultContentLanguageInSubdir": false,
  "languages": {
    "en": {
      "title": "Docsy",
      "description": "Docsy does docs",
      "languageName": "English",
      "weight": 1
    },
    "no": {
      "title": "Docsy",
      "description": "Docsy er operativsystem for skyen",
      "languageName": "Norsk",
      "contentDir": "content/no",
      "time_format_default": "02.01.2006",
      "time_format_blog": "02.01.2006"
    }
  }
}
{{< /tab >}}
{{< /tabpane >}}

Any setting not defined in a `[languages]` block will fall back to the global value for that setting: so, for example, the content directory used for the site above will be `content/en` unless the user selects the Norwegian language option.

Once you've updated your site config, you create a content root directory for each language version in your source repo, such as  `content/en` for English text, and add your [content](/docs/adding-content/content/) as usual. See the [Hugo Docs](https://gohugo.io/content-management/multilingual) on multi-language support for more information.

{{% alert title="Attention (only when using docsy as hugo module)" color="warning" %}}
If you have a multi language installation, please make sure that the section `[languages]` inside your [configuration file](https://gohugo.io/getting-started/configuration/#configuration-file) is declared **before** the section `[module]` with the module imports. Otherwise you will run into trouble!
{{% /alert %}}

{{% alert title="Tip" %}}
If there's any possibility your site might be translated into other languages, consider creating your site with your content in a language-specific subdirectory, as it means you don't need to move it if you add another language.
{{% /alert %}}

For adding multiple language versions of other site elements such as button text, see the [internationalization bundles](#internationalization-bundles) section below.

## Selecting a language

If you configure more than one language in your [configuration file](https://gohugo.io/getting-started/configuration/#configuration-file), the Docsy theme adds a language selector drop down to the top-level menu. Selecting a language takes the user to the translated version of the current page, or the home page for the given language.

## Internationalization bundles

All UI strings (text for buttons, repository links, etc.) are bundled inside `/i18n` in the theme, with a `.toml` file for each language.

If your chosen language isn't currently in the theme and you create your own `.toml` file for all the common UI strings (for example, if you translate the UI text into Esperanto and create a copy of `en.toml` called `eo.toml`), we recommend you do this **in the theme** rather than in your own project. You can then open a [pull request](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request) to contribute your translation to the Docsy community.

{{% alert title="Hugo Tip" %}}
Run `hugo server --printI18nWarnings` when doing translation work, as it will give you warnings on what strings are missing.
{{% /alert %}}

### Create custom UI strings

If any of the Docsy theme UI strings in your chosen language aren't suitable for your project, or if you need additional strings for your site, you can create your own project-specific internationalization file in your project's `/i18n` directory. For example, if you want to override any of Docsy's [English-language strings](https://github.com/google/docsy/blob/main/i18n/en.toml), create your own `/i18n/en.toml` with just your custom strings.  Any values you specify in this file will override the theme versions, while the remaining strings will come from the theme's corresponding internationalization bundle.
