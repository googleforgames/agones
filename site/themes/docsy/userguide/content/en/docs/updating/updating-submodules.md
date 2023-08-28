---
title: "Update Docsy without Hugo Modules"
linkTitle: "Update Docsy without Hugo Modules"
weight: 2
description: >
  Update the Docsy theme to the latest version using submodules or `git pull`.
---

If you aren't using Hugo Modules, depending on how you chose to install Docsy on your existing site, use one of the following two procedures to update your theme.

{{% alert title="Tip" %}}
If you intend to update your site, consider [converting your site to Hugo Modules](/docs/updating/convert-site-to-module/). After conversion, it's even simpler to update Docsy!
{{% /alert %}}

## Update your Docsy submodule

If you are using the [Docsy theme as a submodule](/docs/get-started/other-options/#option-1-docsy-as-a-git-submodule) in your project, here's how you update the submodule:

1. Navigate to the root of your local project, then run:

    ```bash
    git submodule update --remote
    ```
    
1. Add and then commit the change to your project:

    ```bash
    git add themes/
    git commit -m "Updating theme submodule"
    ```

1. Push the commit to your project repo. For example, run:

    ```bash
    git push origin master
    ```

## Route 2: Update your Docsy clone

If you [cloned the Docsy theme](/docs/get-started/other-options/#option-2-clone-the-docsy-theme) into
the `themes` folder in your project, then you use the `git pull` command:

1. Navigate to the `themes` directory in your local project:

    ```bash
    cd themes

1. Ensure that `origin` is set to `https://github.com/google/docsy.git`:

    ```bash
    git remote -v


1. Update your local clone:
    ```bash
    git pull origin master
    ```

If you have made any local changes to the cloned theme, **you must manually resolve any merge conflicts**.
