# Agones Feature Proposals (AFPs)

A Agones Feature Proposal (AFP) is a way to propose, communicate and coordinate on new efforts for the Agones project.
You can read the full details of the project in [AFP-0000](0000-afp-process/README.md).

## Quick start for the AFP process

1. Write a proposal outlining the new feature or improvement you wish to introduce to Agones. Submit your proposal by creating an issue in the Agones GitHub repository, following the [AFP template](NNNN-afp-template/README.md) Make sure that others think the work is worth taking up and will help review the AFP and any code changes required.
2. Follow the process outlined in the [AFP template](NNNN-afp-template/README.md)

## FAQs

### Do I have to use the AFP process?

More or less, yes.

Having a rich set of AFPs in one place will make it easier for people to track
what is going in the community and find a structured historical record.

AFPs are required for most non-trivial changes.  Specifically:
* Anything that may be controversial
* Most new features (except the very smallest)
* Major changes to existing features
* Changes that are wide ranging or impact most of the project (these changes
  are usually coordinated through the relevant maintainers)

Beyond these, it is up to the team to decide when they want to use the AFP 
process. It should be light-weight enough that AFPs are the default position.


### Why would I want to use the AFP process?

Our aim with AFPs is to clearly communicate new efforts to the Agones contributor community.
As such, we want to build a well curated set of clear proposals in a common format with useful metadata.

Benefits to AFP users (in the limit):
* Exposure on a agones blessed web site that is findable via web search engines.
* Cross indexing of AFPs so that users can find connections and the current status of any AFP.
* A clear process with approvers and reviewers for making decisions.
  This will lead to more structured decisions that stick as there is a discoverable record around the decisions.

We are inspired by K8S KEPs, Python PEPs and Rust RFCs.
See [AFP-0000](0000-afp-process/README.md) for more details.

### What will it take for AFPs to "graduate" out of "beta"?

Things we'd like to see happen to consider AFPs well on their way:
* A set of AFPs that show healthy process around describing an effort and recording decisions in a reasonable amount of time.
* AFPs exposed on a searchable and indexable web site.
* Presubmit checks for AFPs around metadata format and markdown validity.

Even so, the process can evolve. As we find new techniques we can improve our processes.

### What is the number at the beginning of the AFP name?

AFPs are now prefixed with their associated tracking issue number. This gives
both the AFP a unique identifier and provides an easy breadcrumb for people to
find the issue where the current state of the AFP is being updated.

### My FAQ isn't answered here!

The AFP process is still evolving!
If something is missing or not answered here feel free to reach out to [Slack](https://join.slack.com/t/agones/shared_invite/zt-2mg1j7ddw-0QYA9IAvFFRKw51ZBK6mkQ).
If you want to propose a change to the AFP process you can open a PR on [AFP-0000](0000-afp-process/README.md) with your proposal.
