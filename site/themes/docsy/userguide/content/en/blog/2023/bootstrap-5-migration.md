---
title: Migrating to Bootstrap 5.2
linkTitle: Bootstrap 5 migration
description: >
  An experience report in migrating Docsy from Bootstrap 4 to 5.2, with insights
  and instructions.
author: >
  [Patrice Chalin](https://github.com/chalin), [CNCF](https://www.cncf.io/) &
  Docsy Steering Committee
date: 2023-06-05
canonical_url: https://www.cncf.io/blog/2023/06/05/migrating-docsy-to-bootstrap-5/
spelling: cSpell:ignore CNCF Chalin opentelemetry techdocs
---

[Docsy](https://docsy.dev), and Docsy-based project websites ([including those
at the CNCF][cncf-docsy]), have been happily using the
[Bootstrap CSS framework](https://getbootstrap.com) from Docsy's inception. In
January of this year, Bootstrap 4 (the version used by Docsy for the past few
years) reached its [end of life](https://endoflife.date/bootstrap). While we,
the Docsy steering committee, have been eager to benefit from the Bootstrap 5
improvements, we were concerned about the magnitude of the migration effort, as
well as the impact on downstream projects. Because of this, the migration was
delayed for as long as possible. In December of 2022, when Bootstrap 4 stopped
receiving critical upgrades, we declared
[Docsy to be in a feature freeze](https://github.com/google/docsy/discussions/1308),
and focused our maintenance efforts on the Bootstrap 5 migration.

This post is about Docsy's migration journey to
[**Bootstrap 5.2**](https://blog.getbootstrap.com/2022/07/19/bootstrap-5-2-0/)[^*]:
it highlights the most notable steps, with a special attention given to the most
surprising aspects of the migration. Our hope is that this post will be useful
to others upgrading to Bootstrap 5, in particular, for downstream Docsy projects
--- though we plan a separate post specifically for downstream projects.

## TL;DR

Eager to dive into the Bootstrap migration of your project? Besides carefully
stepping through the
[Bootstrap migration page](https://getbootstrap.com/docs/5.2/migration/), watch
out for the following:

- The `media-breakpoint-down()` mixin's breakpoint argument needs to be shifted.
- Grid `.row` and `.col` style changes are breaking.
- Import ordering of Bootstrap Sass files: import functions first.

For details, read on.

## Technical details

If you are well accustomed to upgrading Docsy (and its dependencies) by reading
changelogs and systematically stepping through commits, then this section
provides a summary of some notable changes. In it, I describe technical aspects
of the migration that surprised me, either because they required particular care
in fixing, were undocumented, and/or insufficiently explained in the Bootstrap
[migration page](https://getbootstrap.com/docs/5.2/migration/).

### Mixin `media-breakpoint-down()` argument shift

The [breakpoint](https://getbootstrap.com/docs/5.2/layout/breakpoints) argument
to the `media-breakpoint-down()` mixin needs to be bumped up to the next higher
breakpoint. Thankfully, a similar change isn't required of
`media-breakpoint-up()`. This change will be required of Docsy-based projects.
If you forget to make this non-obviously breaking layout change, your project's
responsive layouts will likely start misbehaving in apparently strange ways.

For details and an example, see:

- [Sass](https://getbootstrap.com/docs/5.2/migration/#sass) section of the
  migration page
- [\[BSv5\] Adjust `media-breakpoint-down()` argument · Docsy PR #1367](https://github.com/google/docsy/pull/1367)

### Grid `.row` and `.col` style changes are breaking

The main issue addressed in this section is not, at the time of writing,
documented in the Bootstrap 5
[migration page](https://getbootstrap.com/docs/5.2/migration).

There seems to be an assumption, in Bootstrap 5, that the immediate child of a
.`row` should be a `.col`. I don't know how strict an assumption this is. While
I have searched for an explicit statement of this assumption in the Bootstrap
documentation, I haven't found one yet --- if you are aware of such a statement,
let us know!

This assumption wasn't apparent nor was it enforced in Bootstrap 4,
consequently, some of Docsy's layouts failed to respect it. In
[most cases](https://github.com/google/docsy/issues/1466), fixing violations
consisted of simply wrapping a `.row`'s child element in a `.col`, but the
[Docsy footer](http://layouts/partials/footer.html) required a couple of
iterations to get right.

My first footer adjustment reset
[`flex-shrink`](https://developer.mozilla.org/en-US/docs/Web/CSS/flex-shrink) to
its default value (PR [#1373](https://github.com/google/docsy/pull/1373)), but
that turned out to be unnecessary once I better understood how to appropriately
handle row margins (PR [#1523](https://github.com/google/docsy/pull/1523)) ---
rows have negative margins, as I
[recently learned](https://github.com/google/docsy/pull/1502#issue-1678874640),
which is something to keep in mind.

The following Bootstrap 5 `.col` style changes influenced Docsy-specific style
updates and might impact Docsy-based projects as well:

- `position` is reverted to its
  [default value of `static`](https://developer.mozilla.org/en-US/docs/Web/CSS/position#values)
  from
  [`relative`](https://github.com/twbs/bootstrap/pull/28517/files#diff-41667d8b9901aa9fa52483b538bb9026c287f2c663d2fdc01acffa06888cc087L13)
- `flex-shrink`
  [default value of 1](https://developer.mozilla.org/en-US/docs/Web/CSS/flex-shrink#values)
  is overridden and
  [set to 0](https://github.com/twbs/bootstrap/pull/28517/files#diff-41667d8b9901aa9fa52483b538bb9026c287f2c663d2fdc01acffa06888cc087R18)

References:

- [\[BSv5\] Row/col formatting breaks Docsy components #1466](https://github.com/google/docsy/issues/1466),
  in particular
  - [\[BSv5\] Footer fixes: reset flex-shrink, and more·](https://github.com/google/docsy/pull/1373)
    [Docsy PR](https://github.com/google/docsy/pull/1367)[ #1373](https://github.com/google/docsy/pull/1373)
  - [\[BSv5\] Footer: drop flex-shrink tweak + other adjustments ·](https://github.com/google/docsy/pull/1523)
    [Docsy PR](https://github.com/google/docsy/pull/1367)[ #1523](https://github.com/google/docsy/pull/1523)
- [Why are all col classes 'position: relative'? · Bootstrap v4 issue #25254](https://github.com/twbs/bootstrap/issues/25254)
- [Why flex-shrink 0 for all columns? · Bootstrap discussion #37951](https://github.com/orgs/twbs/discussions/37951)

### Import ordering of Bootstrap Sass files: functions first

Projects can [import](https://getbootstrap.com/docs/5.2/customize/sass/)
Bootstrap Sass sources all in one go (using
[bootstrap.scss](https://github.com/twbs/bootstrap/blob/v5.2.3/scss/bootstrap.scss)),
or selectively import any one of the
[40+](https://github.com/twbs/bootstrap/blob/v5.2.3/scss/bootstrap.scss)
Bootstrap parts, layouts, and components that they need. Regardless of the
import strategy chosen, due to a Sass map initialization limitation,
Bootstrap-client projects need to perform (emphasis mine):

> ... variable customizations ... **after** `@import "functions"`, but
> **before** > `@import "variables"` and the rest of [the Bootstrap] import
> stack.

For details, see
[New \_maps.scss](https://getbootstrap.com/docs/5.2/migration/#new-_mapsscss)
from the migration page, and
[Importing](https://getbootstrap.com/docs/5.2/customize/sass/) from Bootstrap's
Sass customization documentation.

Having to maintain a custom list of a few dozen imports (even if it's relatively
stable) feels like a maintenance overhead that we should avoid if we can, so in
Docsy's
[main.scss](https://github.com/google/docsy/blob/v5.2.3/assets/scss/main.scss),
we \@import "functions" before Docsy- and project-specific variable overrides,
and then we import the _full_ Bootstrap suite of SCSS. This results in
[\_functions.scss](https://github.com/twbs/bootstrap/blob/v5.2.3/scss/_functions.scss)
being imported twice, but according to the
[Sass `@import` documentation](https://sass-lang.com/documentation/at-rules/import):

> If the same stylesheet is imported more than once, it will be evaluated again
> each time. If it just defines functions and mixins, this usually isn't a big
> deal, but if it contains style rules they'll be compiled to CSS more than
> once.

The
[\_functions.scss](https://github.com/twbs/bootstrap/blob/v5.2.3/scss/_functions.scss)
file only contains function definitions, so we should be ok. This seems like a
small cost to pay in contrast to the alternative strategy of inlining the 40+
imports from
[bootstrap.scss](https://github.com/twbs/bootstrap/blob/v5.2.3/scss/bootstrap.scss).

References:

- [\[BSv5\] Fix SCSS functions import issue ... ·](https://github.com/google/docsy/pull/1388)
  [Docsy PR](https://github.com/google/docsy/pull/1367)
  [#1388](https://github.com/google/docsy/pull/1388)
- [New \_maps.scss](https://getbootstrap.com/docs/5.2/migration/#new-_mapsscss)
  from the migration page
- [Importing](https://getbootstrap.com/docs/5.2/customize/sass/) from
  Bootstrap's Sass customization documentation

## Systematic and stepwise migration

If you've glanced at the Bootstrap 5
[migration page](https://getbootstrap.com/docs/5.2/migration/), you will see
that there are a _lot_ of changes to address while migrating. To ensure that we
didn't miss any, we systematically walked through the migration guide, and
tracked the status of each change through
[Docsy issue #470](https://github.com/google/docsy/issues/470). Each relevant
migration page section is represented in the issue's opening comment: we either
noted that a migration-page section is irrelevant for Docsy, or added the
section to the tracking issue, and list the PRs containing corresponding
Docsy-specific changes. If you're curious to see how that worked out, see
[Upgrade to Bootstrap 5.2 · Docsy issue #470](https://github.com/google/docsy/issues/470).

## First Bootstrap 5 release of Docsy

A first Bootstrap 5 release of Docsy is planned for the start of June, since
most aspects of the migration have been completed. Some updates have been
postponed, most notably support for right-to-left
([RTL](https://getbootstrap.com/docs/5.2/migration/#rtl)) text. For the complete
list of followup items, see
[BSv5.2 upgrade followup · Docsy issue #1510](https://github.com/google/docsy/issues/1510).

As was mentioned earlier, this first release will be in support of
[Bootstrap 5.2](https://blog.getbootstrap.com/2022/07/19/bootstrap-5-2-0/). We
plan a separate migration effort to bring Docsy up to
[Bootstrap 5.3](https://blog.getbootstrap.com/2023/05/30/bootstrap-5-3-0/), in
particular to benefit from new
[color modes](https://blog.getbootstrap.com/2023/05/30/bootstrap-5-3-0/#custom-color-modes).
You can track our progress through
[Docsy issue #1528](https://github.com/google/docsy/issues/1528).

## Migrating Docsy-based projects

This section contains some preliminary and general guidance for downstream
projects. We are planning a separate post to cover more migration details.

### Bootstrap migration-page walkthrough

Each project uses its own specific set of Bootstrap features, so walking through
the Bootstrap 5.2 [migration page](https://getbootstrap.com/docs/5.2/migration/)
will be advisable for most projects. Of course, one strategy is just to upgrade
and see what breaks or no longer works, but only doing that without a more
systematic follow-up would be ill-advised for all but the most trivial
projects---consider the challenge in detecting and recovering from a missed
change to a ​​`media-breakpoint-down()` argument, as discussed earlier.

### Docsy-specific changes

During the migration effort we seized the opportunity to do some long overdue
Docsy house cleaning. For details concerning both breaking and non-breaking
Docsy-specific changes, consult the
[changelog](https://github.com/google/docsy/blob/main/CHANGELOG.md#070). In
particular, one non-breaking but important change to be aware of is:
[\[BSv5\] Docsy variables cleanup ... PR #1462](https://github.com/google/docsy/pull/1462).

## Give it a try!

To get a first and quick impression of the impact of the upgrade on your
project, it can be informative to simply upgrade Docsy and see what breaks. This
is what the Docsy team did with Bootstrap 5. Only one change actually broke the
build of the Docsy User Guide: the
[rename of the `color-yiq()` function](https://getbootstrap.com/docs/5.2/migration/#sass).

After such a smoke test, we recommend systematically walking through the
Bootstrap [migration page](https://getbootstrap.com/docs/5.2/migration/) as
described above, and the Docsy
[changelog](https://github.com/google/docsy/blob/main/CHANGELOG.md#070). I used
this approach for [opentelemetry.io](https://opentelemetry.io/), which was the
first Docsy-based project to be upgraded with a pre-release of Bootstrap-5-based
Docsy. The upgrade went
[quite smoothly](https://github.com/open-telemetry/opentelemetry.io/issues/2419).
The main pain point of the OTel website was upgrading to Bootstrap 5
[forms](https://getbootstrap.com/docs/5.2/migration/#forms); an aspect of the
migration that didn't apply to Docsy since Docsy uses only the most trivial of
forms.

We'll have more to share about the OTel migration effort as well as general
project-specific migration advice in a followup blog post. In the meantime, I
hope that you have found parts of this technical article helpful for your own
migration efforts.

[CNCF project](https://www.cncf.io/projects/) websites eager to migrate can send
questions to the CNCF
[#techdocs Slack channel](https://cloud-native.slack.com/archives/CUJ6W5TLM).
CNCF and other Docsy-based projects can also
[start a discussion](https://github.com/google/docsy/discussions/new) in the
Docsy repository. Happy migrating!

A big thanks to the Docsy Steering Committee and other reviewers who offered
feedback on earlier drafts of this post, as well as to all those who contributed
to the migration effort.

[^*]:
    [Bootstrap 5.3 reached GA](https://blog.getbootstrap.com/2023/05/30/bootstrap-5-3-0/)
    on May 30. There will be a separate migration effort to bring
    [Docsy up to Bootstrap 5.3](https://github.com/google/docsy/issues/1528).

_A version of this article originally appeared as the [CNCF blog][] post
[Migrating Docsy to Bootstrap 5 ][original post]._

[cncf blog]: https://www.cncf.io/blog/
[cncf-docsy]:
  https://www.cncf.io/blog/2023/01/19/fast-and-effective-tools-for-cncf-and-open-source-project-websites/

[original post]: {{% param canonical_url %}}
