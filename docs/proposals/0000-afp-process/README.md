# Agones Feature Proposal Process

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
- [Stewardship](#stewardship)
- [Reference-Level Explanation](#reference-level-explanation)
  - [What Type of Work Should Be Tracked by a AFP](#what-type-of-work-should-be-tracked-by-a-afp)
  - [AFP Template](#afp-template)
  - [AFP Metadata](#afp-metadata)
  - [AFP Workflow](#afp-workflow)
  - [Git and GitHub Implementation](#git-and-github-implementation)
  - [AFP Editor Role](#afp-editor-role)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [GitHub Issues vs. AFPs](#github-issues-vs-afps)
<!-- /toc -->

## Summary

A standardized development process for Agones is proposed, in order to:

- provide a common structure for proposing changes to Agones
- ensure that the motivation for a change is clear
- persist project information in a Version Control System (VCS) for future
  Agones
- support the creation of _high-value, user-facing_ information such as:
  - an overall project development roadmap
  - motivation for impactful user-facing changes
- reserve GitHub issues for tracking work in flight, instead of creating "umbrella"
  issues
- enable community participants to drive changes effectively across multiple releases 
  while ensuring stakeholder representation throughout the process.

This process is supported by a unit of work called a Agones Feature Proposal, or AFP.
A AFP attempts to combine aspects of

- a feature, and effort-tracking document
- a product requirements document
- a design document

into one file, which is created incrementally through collaboration with relevant stakeholders and teams.

## Motivation

Currently, design discussions in Agones primarily occur as comments on GitHub issues. 
While this provides flexibility, it can also lead to confusion and inefficiencies. 
Consider the discussion in issue [#3882][] as an example:

The design was introduced as a follow-up comment rather than in the original post, 
making it harder to track. To improve clarity, the first comment was updated to link 
to the design discussion, but the thread remained busy and somewhat disorganized.

In other cases, we create a new issue with the design and close the related issues, 
which ensures the design appears at the top but fragments discussions.

Version control is lacking, making it difficult to track iterations. For instance, 
updates to the design might render prior comments irrelevant, leading to disjointed 
conversations.Providing feedback on lengthy designs requires quoting sections manually 
instead of allowing inline commenting.

To address these challenges, we propose adopting a structured process for feature 
proposals, similar to Kubernetes' KEP process. This would introduce a lightweight 
mechanism called Agones Feature Proposals (AFP):

A AFP is broken into sections which can be merged into source control
incrementally in order to support an iterative development process. An important
goal of the AFP process is ensuring that the process for submitting the content
contained in [design proposals][] is both clear and efficient. The AFP process
is intended to create high-quality, uniform design and implementation documents 
to support deliberation within the project.

[design proposals]: /docs/proposals/NNNN-afp-template
[#3882]: https://github.com/googleforgames/agones/issues/3882

## Stewardship
The following DACI model identifies the responsible parties for AFPs.

**Workstream** | **Driver** | **Approver** | **Contributor** | **Informed**
--- | --- | --- | --- | ---
| AFP Process Stewardship | - | - |  Project Maintainers | Community |
| Proposal Delivery | Proposal Owner |  Project Maintainers  | Proposal Implementer(s) (may overlap with Driver) | Community |


## Reference-Level Explanation

### What Type of Work Should Be Tracked by a AFP

The definition of what constitutes a "feature" is a foundational concern
for the Agones project. Roughly any Agones user or operator facing
feature should follow the AFP process. If a feature would be described
in either written or verbal communication to anyone besides the AFP author or
developer, then consider creating a AFP.

Similarly, any technical effort (refactoring, major architectural change) that
will impact a large section of the development community should also be
communicated widely. The AFP process is suited for this even if it will have
zero impact on the typical user or operator.

As the local bodies of governance, project teams or contributors should have broad latitude in describing
what constitutes an feature that should be tracked through the AFP process.
It may be more helpful for these teams to enumerate what _does not_ require a AFP,
than what does. Teams also have the freedom to customize the AFP template
according to their specific concerns. For example, the AFP template used to
track API changes will likely have different subsections than the template for
proposing governance changes. However, as changes start impacting other areas or
the larger developer community, the AFP process should be used to coordinate and communicate.

Features that have major impacts on multiple areas of the Agones should use the AFP process.
AFPs will also be used to drive large changes that will cut across all parts of the project.
These AFPs will be managed by the core team and should be regarded as the primary means of communicating the most fundamental aspects of what Agones is.


### AFP Template

**The template for a AFP is precisely defined [here](/docs/proposals/NNNN-afp-template).**

### AFP Metadata

There is a place in each AFP for a YAML document that has standard metadata.
This will be used to support tooling around filtering and display. It is also
critical to clearly communicate the status of a AFP.

<b>
While this defines the metadata schema for now, these things tend to evolve.
The AFP template is the authoritative definition of things like the metadata
schema.
</b>

Metadata items:
* **title** Required
  * The title of the AFP in plain language. The title will also be used in the
    AFP filename.  See the template for instructions and details.
* **status** Required
  * The current state of the AFP.
  * Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
* **authors** Required
  * A list of authors of the AFP.
    This is simply the GitHub ID.
    In the future we may enhance this to include other types of identification.
* **reviewers** Required
  * Reviewer(s) chosen after triage, according to the proposal process.
  * Reviewer(s) selected based on the impacted areas or components of the project.
    It is up to the individual areas to determine how they pick reviewers for AFPs impacting them.
    The reviewers are speaking for the respective areas in the process of approving this AFP
    The impacted areas can modify this list as necessary.
  * The reviewers are the individuals who decide when to move this AFP to the `implementable` state.
  * If not yet chosen, replace with `TBD`.
  * Same name/contact scheme as `authors`.
* **editor** Required
  * Someone to keep things moving forward.
  * If not yet chosen, replace with `TBD`.
  * Same name/contact scheme as `authors`.
* **creation-date** Required
  * The date that the AFP was first submitted in a PR.
  * In the form `yyyy-mm-dd`.
  * While this info will also be in source control, it is helpful to have the set of AFP files stand on their own.
* **last-updated** Optional
  * The date that the AFP was last changed significantly.
  * In the form `yyyy-mm-dd`.
* **see-also** Optional
  * A list of other AFPs that are relevant to this AFP.
  * In the form `AFP-123`.
* **replaces** Optional
  * A list of AFPs that this AFP replaces. Those AFPs should list this AFP in
    their `superseded-by`.
  * In the form `AFP-123`.
* **superseded-by**
  * A list of AFPs that supersede this AFP. Use of this should be paired with
    this AFP moving into the `Replaced` status.
  * In the form `AFP-123`.


### AFP Workflow

A AFP has the following states:

- `provisional`: The AFP has been proposed and is actively being defined.
  This is the starting state while the AFP is being fleshed out and actively defined and discussed.
  The maintainer has accepted that this work must be done.
- `implementable`: The reviewers have approved this AFP for implementation.
- `implemented`: The AFP has been implemented and is no longer actively changed.
- `deferred`: The AFP is proposed but not actively being worked on.
- `rejected`: The reviewers and authors have decided that this AFP is not moving forward.
  The AFP is kept around as a historical document.
- `withdrawn`: The authors have withdrawn the AFP.
- `replaced`: The AFP has been replaced by a new AFP.
  The `superseded-by` metadata value should point to the new AFP.

### Git and GitHub Implementation

AFPs are checked into the repository under the `/docs/proposals` directory.

New AFPs can be checked in with a file name in the form of `00x-feature-name.md`.
The corresponding PR should be titled in the format `AFP-00x`, where 00x represents the AFP number.
As significant work is done on the AFP, the authors can assign a AFP number.
No other changes should be put in that PR so that it can be approved quickly and minimize merge conflicts.
The AFP number can also be done as part of the initial submission if the PR is likely to be uncontested and merged quickly.

### AFP Editor Role

Taking a cue from the [Kubernetes KEP process][] & [Python PEP process][], we define the role of a AFP editor.
The job of an AFP editor is likely very similar to the [PEP editor responsibilities][] and will hopefully provide another opportunity for people who do not write code daily to contribute to Agones.

In keeping with the PEP editors, who:

> Read the PEP to check if it is ready: sound and complete. The ideas must make
> technical sense, even if they don't seem likely to be accepted.
> The title should accurately describe the content.
> Edit the PEP for language (spelling, grammar, sentence structure, etc.), markup
> (for reST PEPs), code style (examples should match PEP 8 & 7)

AFP editors should generally not pass judgement on a AFP beyond editorial corrections.
AFP editors can also help inform authors about the process and otherwise help things move smoothly.

[Kubernetes KEP process]: https://github.com/kubernetes/enhancements/tree/master/keps
[Python PEP process]: https://www.python.org/dev/peps/pep-0001/
[PEP editor responsibilities]: https://www.python.org/dev/peps/pep-0001/#pep-editor-responsibilities-workflow

## Drawbacks

Adding more steps to the process might frustrate people in the community. 
There's also a chance that the AFP process won’t solve our scaling problems. 
Since PR reviews already take a lot of time, and we may find that the AFP process 
introduces an unreasonable bottleneck on our development velocity.

It certainly can be argued that the lack of a dedicated issue/defect tracker
beyond GitHub issues contributes to our challenges in managing a project as large
as Agones. However, given that other large organizations, including GitHub
itself, make effective use of GitHub issues, perhaps the argument is overblown.

The centrality of Git and GitHub within the AFP process also may place too high
a barrier to potential contributors. However, given that both Git and GitHub are
required to contribute code changes to Agones today, perhaps it would be reasonable
to invest in providing support to those unfamiliar with this tooling.

Expanding the proposal template beyond the single-sentence description currently
required in the [features issue template][] may be a heavy burden for non-native
English speakers. Here, the role of the AFP editor, combined with kindness and
empathy, will be crucial to making the process successful.

[features issue template]: https://github.com/googleforgames/agones/blob/main/.github/ISSUE_TEMPLATE/feature_request.md

### GitHub Issues vs. AFPs

The use of GitHub issues when proposing changes does not provide an effective mechanism good
facilities for signaling approval or rejection of a proposed change to Agones,
because anyone can open a GitHub issue at any time. Additionally, managing a proposed
change across multiple releases is somewhat cumbersome as labels and milestones
need to be updated for every release that a change spans. These long-lived GitHub
issues lead to an ever-increasing number of issues open against
`agones/docs/proposals`, which itself has become a management problem.

In addition to the challenge of managing issues over time, searching for text
within an issue can be challenging. The flat hierarchy of issues can also make
navigation and categorization tricky. Not all community members will
be uncomfortable using Git directly, but it is imperative for our community to educate people on a standard set of tools so they can take their
experience to other projects they may decide to work on in the future. While
git is a fantastic version control system (VCS), it is neither a project management
tool nor a cogent way of managing an architectural catalog or backlog. This
proposal is limited to motivating the creation of a standardized definition of
work in order to facilitate project management. This primitive for describing
a unit of work may also allow contributors to create their own personalized
view of the state of the project while relying on Git and GitHub for consistency
and durable storage.