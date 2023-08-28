---
title: Update your Docsy Hugo Module
weight: 1
description: Update your Docsy theme to the latest version using Hugo Modules.
---

When using the Docsy theme as a Hugo Module, updating your theme is really easy.

At the command prompt, change to the root directory of your existing site.

```bash
cd /path/to/my-existing-site
```

Then invoke hugo's module `get` subcommand with the update flag:

```bash
hugo mod get -u github.com/google/docsy
```

Hugo automatically pulls in the latest theme version. That's it, your update is
done!

{{% alert title="Tip" %}}

If you want to set your module to a certain version inside the Docsy theme repo,
simply specify the name of the tag representing this version when updating your
theme, for example:

```bash
hugo mod get -u github.com/google/docsy@v{{% param "version" %}}
```

Instead of a version tag, you can also specify a commit hash, for example:

```bash
hugo mod get -u github.com/google/docsy@6c8a3afe
```

{{% /alert %}}
