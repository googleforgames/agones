# Agones Release Management

# Release Cadence

- Versioned releases will occur every 6 weeks
- Releases occur on a Tuesday.
- 5 week development cycle, at the end of a which a Release Candidate (RC) will be released with the contents of master.
- For the next week, the project is in "feature freeze" - i.e. only bug and documentation (.md and examples) fixes during this time.
  - Any new PRs that are submitted during feature freeze, will be tagged with the label `merge-after-release` 
    to delineate that they should only be merged after the full release is complete. 
- At the end of the RC week, the complete version release will occur.

## Release Calendar

- [Web View](https://calendar.google.com/calendar/embed?src=google.com_828n8f18hfbtrs4vu4h1sks218%40group.calendar.google.com&ctz=America%2FLos_Angeles)
- [iCal](https://calendar.google.com/calendar/ical/google.com_828n8f18hfbtrs4vu4h1sks218%40group.calendar.google.com/public/basic.ics)

# Release Process

1. Create a Release Issue from the [release issue template](./templates/release_issue.md).
1. Label the issue `kind/release`, and attach it to the milestone that it matches.
1. Complete all items in the release issue checklist.
1. Close the release issue.

# Hot fix Process
 
1. Hotfixes will occur as needed, to be determined by those will commit access on the repository.
1. Create a Release Issue from the [release issue template](./templates/release_issue.md).
1. Label the issue `release`, and attach it to the next upcoming milestone.
1. Complete all items in the release issue checklist.
1. Close the release issue.


