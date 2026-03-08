# Review Criteria

Secret, repo-specific review criteria. Only Armitage reads this file.

## Mission Criteria

### What This Repo MUST Achieve

- Template must produce a repo that passes CI on first push
- No placeholder content in user-facing files
- All infrastructure is functional, not decorative

### What This Repo MUST NOT Contain

- No hardcoded zoobzio-specific values that template users must find-and-replace
- No dependencies beyond stdlib
- No application logic — infrastructure only

## Review Priorities

Ordered by importance. When findings conflict, higher-priority items take precedence.

1. Security: no secrets, no unsafe defaults, no vulnerable dependencies
2. Correctness: CI workflows actually work, linter config is valid, coverage config is correct
3. Completeness: all template infrastructure present and functional
4. Quality: conventions are consistent, documentation is accurate

## Severity Calibration

Guidance for how Armitage classifies finding severity for this specific repo.

| Condition | Severity |
|-----------|----------|
| Missing or broken CI workflow | Critical |
| Security vulnerability in dependencies | Critical |
| Linter config references nonexistent rules | High |
| Coverage workflow fails to merge profiles | High |
| Incorrect badge URL | Medium |
| Missing benchmark README content | Low |
| Minor inconsistency in template comments | Low |

## Standing Concerns

Persistent issues or areas of known weakness that should always be checked.

- Coverage workflow has historically failed to merge profiles correctly
- Release workflow submodule tagging sequence is fragile
- golangci-lint v2 migration may leave stale v1 config patterns

## Out of Scope

Things the red team should NOT flag for this repo, even if they look wrong.

- api.go is intentionally minimal — it is a template placeholder
- README is intentionally sparse — template users replace it
- MISSION.md references "samoa" specifically — this is correct for the template itself
