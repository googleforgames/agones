# Community Membership

This document outlines the responsibilities of contributor roles in Agones.

This is based on the [Kubernetes Community Membership](https://github.com/kubernetes/community/blob/master/community-membership.md).

There are currently only one role for this project, but that may grow in the future.

| Role | Responsibilities | Requirements | Defined by |
| -----| ---------------- | ------------ | -------|
| approver | review and approve contributions | sponsored by 2 reviewers.  multiple contributions to the project. | Commit access to the Agones repository |

## New contributors

New contributors should be welcomed to the community by existing members,
helped with PR workflow, and directed to relevant documentation and
communication channels.

## Established community members

Established community members are expected to demonstrate their adherence to the
principles in this document, familiarity with project organization, roles,
policies, procedures, conventions, etc., and technical and/or writing ability.
Role-specific expectations, responsibilities, and requirements are enumerated
below.

## Approvers

Code approvers are able to both review and approve code contributions.  While
code review is focused on code quality and correctness, approval is focused on
holistic acceptance of a contribution including: backwards / forwards
compatibility, adhering to API and flag conventions, subtle performance and
correctness issues, interactions with other parts of the system, etc.

**Defined by:** Commit access to the Agones repository.

**Note:** Acceptance of code contributions requires at least one approver.

### Requirements

- Enabled [two-factor authentication](https://help.github.com/articles/about-two-factor-authentication)
  on their GitHub account
- Have made multiple contributions to Agones.  Contribution must include:
    - Authored at least 3 PRs on Github
    - Provided reviews on at least 4 PRs they did not author
    - Filing or commenting on issues on GitHub
- Have read the [contributor guide](../../CONTRIBUTING.md)
- Sponsored by 2 approvers. **Note the following requirements for sponsors**:
    - Sponsors must have close interactions with the prospective member - e.g. code/design/proposal review, coordinating
      on issues, etc.
    - Sponsors must be from multiple companies to demonstrate integration across community.
- **[Open an issue](./templates/membership.md) against the Agones repo**
   - Ensure your sponsors are @mentioned on the issue
   - Label the issue with the `meta` tag
   - Complete every item on the checklist ([preview the current version of the template](./templates/membership.md))
   - Make sure that the list of contributions included is representative of your work on the project.
- Have your sponsoring approvers reply confirmation of sponsorship: `+1`

### Responsibilities and privileges

- Responsible for project quality control via code reviews
  - Focus on code quality and correctness, including testing and factoring
  - May also review for more holistic issues, but not a requirement
- Expected to be responsive to review requests in a timely manner
- Assigned PRs to review related based on expertise
- Granted commit access to Agones repo
