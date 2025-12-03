# Contributing

Thanks for contributing! This document describes the development workflow and branch/PR conventions we use so history stays clean and changes are easy to review and revert.

## Branching & branch names
- Create a new branch for every logical change (feature, bugfix, refactor, docs).
- Branches should be short-lived and single-purpose. Do not reuse topic branches for multiple unrelated changes.
- Naming convention:
  `<type>/<area>/<short-desc>-<issue>`
  - `type`: `feat` | `fix` | `refactor` | `chore` | `docs` | `test`
  - `area`: the subsystem (e.g. `backend`, `proxy`, `cache`)
  - `short-desc`: kebab-case summary
  - `issue`: optional issue or ticket number
- Examples:
  - `feat/backend/round-robin-lb-1234`
  - `fix/backend/cache-key-567`
  - `refactor/proxy/handler-simplify-890`

Why: branches scoped to a single task make reviews, CI, and rollbacks much easier.

## Workflow (recommended)
1. Update your local `master`:
   - `git checkout master`
   - `git pull origin master`
2. Create a branch:
   - `git checkout -b feat/<area>/<short-desc>-<issue>`
3. Work, run tests locally, make atomic commits.
4. Push the branch:
   - `git push -u origin <branch>`
5. Open a PR against `master` with a descriptive title and description.
6. Address review feedback in the same branch (force-push if you rebase).
7. Rebase onto latest `master` or merge `master` into your branch before final approval.
8. Merge via the PR UI when CI passes and reviewers approve.
9. Delete the branch after merge (both remote and local).

## Commit messages
- Use Conventional Commits style: `<type>(<scope>): <short summary>`
  - Example: `feat(backend): add round-robin load balancer`
- Keep commits focused and atomic.
- Provide a longer body when necessary to explain reasoning, trade-offs, or design decisions.

## Pull Request checklist (PR template)
Before requesting review, ensure:
- [ ] Branch is scoped to a single logical change.
- [ ] All tests pass locally.
- [ ] New behavior is covered by tests where appropriate.
- [ ] CI passes for the branch.
- [ ] Description explains the motivation and the change.
- [ ] Link to the related issue/ticket (if applicable).
- [ ] Any necessary migration steps or breaking changes are documented.

## CI and tests
- CI runs the test suite and static checks on every push.
- Ensure tests are fast and deterministic. If adding integration tests, mark them separately or use CI stages so they don't block fast feedback.

## Code review
- Keep PRs small and focused when possible.
- Add targeted reviewers (those who own the area changed).
- Respond to comments promptly and push follow-ups to the same branch.
- For larger features, consider splitting into multiple PRs: API, core logic, integration.

## Merging
- Prefer `Squash and merge` or `Rebase and merge` to keep `master` linear and readable.
- Use merge commits only when you need to preserve branch history intentionally.
- After merging:
  - `git checkout master`
  - `git pull`
  - `git branch -d <branch>`
  - `git push origin --delete <branch>`

## Long-lived topic branches
- Avoid long-lived shared topic branches unless absolutely necessary.
- If a team must collaborate on a long-lived branch, consider creating short-lived feature branches that merge into the topic branch, and open PRs against the topic branch — then merge the topic branch when the work is complete.

## Extras (optional)
- Enforce branch name patterns with a CI check or pre-push hook.
- Use Conventional Commits to auto-generate changelogs.
- Protect `master`: require PRs to pass CI and at least one reviewer approval.
