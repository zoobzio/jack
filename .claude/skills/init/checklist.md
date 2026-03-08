# Init Checklist

## Phase 1: Gather Inputs

- [ ] Ask user for package name
- [ ] Ask user for one-line description
- [ ] Ask user for purpose statement (one sentence for MISSION.md)
- [ ] Validate package name: lowercase, no hyphens, valid Go identifier (`^[a-z][a-z0-9]*$`)

## Phase 2: Go Infrastructure

Replace `samoa` with the package name in:

- [ ] `go.mod` — module path `github.com/zoobzio/{name}`
- [ ] `api.go` — package declaration (`package {name}`) and doc comment
- [ ] `testing/helpers.go` — package doc comment
- [ ] `.github/workflows/release.yml` — install command path

## Phase 3: Configuration

Replace `samoa` with the package name in:

- [ ] `.goreleaser.yml` — `project_name` field and `name` field
- [ ] `.github/settings.yml` — `name`, `description` (use one-liner), `homepage` URL
- [ ] `Makefile` — help text echo line

## Phase 4: Documentation

Replace `samoa` with the package name in:

- [ ] `CONTRIBUTING.md` — title and body references
- [ ] `SECURITY.md` — body reference
- [ ] `testing/README.md` — body text and import path
- [ ] `testing/benchmarks/README.md` — body reference
- [ ] `testing/integration/README.md` — body reference

## Phase 5: MISSION.md

Rewrite `.claude/MISSION.md` entirely:

```markdown
# Mission: {name}

{purpose statement}

## Purpose

{purpose statement}

## What This Package Contains

<!-- Populate this section as you build the package. -->
<!-- List the concrete capabilities this package provides. -->

## What This Package Does NOT Contain

<!-- Define boundaries. What is explicitly out of scope? -->

## Success Criteria

<!-- What must be true for this package to be considered complete? -->
<!-- Use numbered, testable criteria. -->

## Non-Goals

<!-- What will this package never do, even if asked? -->
```

## Phase 6: CRITERIA.md

Rewrite `.claude/CRITERIA.md` entirely:

```markdown
# Review Criteria

Secret, repo-specific review criteria. Only Armitage reads this file.

## Mission Criteria

### What This Repo MUST Achieve

<!-- Define the non-negotiable requirements for this package. -->

### What This Repo MUST NOT Contain

<!-- Define hard boundaries that reviewers should flag as violations. -->

## Review Priorities

Ordered by importance. When findings conflict, higher-priority items take precedence.

<!-- Numbered list of review priorities, from most to least important. -->
<!-- Example: -->
<!-- 1. Security: no secrets, no unsafe defaults -->
<!-- 2. Correctness: public API behaves as documented -->

## Severity Calibration

Guidance for how Armitage classifies finding severity for this specific repo.

| Condition | Severity |
|-----------|----------|
| <!-- Describe condition --> | <!-- Critical/High/Medium/Low --> |

## Standing Concerns

Persistent issues or areas of known weakness that should always be checked.

<!-- List any known fragile areas, historical issues, or recurring problems. -->

## Out of Scope

Things the red team should NOT flag for this repo, even if they look wrong.

<!-- List intentional deviations that reviewers should ignore. -->
```

## Phase 7: README.md

Replace `README.md` entirely with:

```markdown
# {name}

[![CI Status](https://github.com/zoobzio/{name}/workflows/CI/badge.svg)](https://github.com/zoobzio/{name}/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/zoobzio/{name}/graph/badge.svg?branch=main)](https://codecov.io/gh/zoobzio/{name})
[![Go Report Card](https://goreportcard.com/badge/github.com/zoobzio/{name})](https://goreportcard.com/report/github.com/zoobzio/{name})
[![CodeQL](https://github.com/zoobzio/{name}/workflows/CodeQL/badge.svg)](https://github.com/zoobzio/{name}/security/code-scanning)
[![Go Reference](https://pkg.go.dev/badge/github.com/zoobzio/{name}.svg)](https://pkg.go.dev/github.com/zoobzio/{name})
[![License](https://img.shields.io/github/license/zoobzio/{name})](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/zoobzio/{name})](go.mod)
[![Release](https://img.shields.io/github/v/release/zoobzio/{name})](https://github.com/zoobzio/{name}/releases)

{one-line description}
```

## Phase 8: Verify

- [ ] Run `go build ./...` — must pass
- [ ] Run `go vet ./...` — must pass
- [ ] Grep for remaining `samoa` references — must find none
- [ ] Present summary: files changed, package name, next steps

## Phase 9: Self-Remove

- [ ] Delete `.claude/skills/init/SKILL.md`
- [ ] Delete `.claude/skills/init/checklist.md`
- [ ] Remove `.claude/skills/init/` directory
