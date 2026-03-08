# Initialize Package

Transform the samoa template into a new package. This skill is scaffolding — it runs once and removes itself.

## Principles

1. **Complete transformation** — Every `samoa` reference becomes the new package name. No orphaned template references.
2. **Blank slate, not guesswork** — MISSION.md and CRITERIA.md get blank templates with guidance, not invented content.
3. **Verify before declaring done** — The package must compile and contain zero `samoa` references when finished.

## Execution

1. Read `checklist.md` in this skill directory
2. Gather inputs from user: package name, one-line description, purpose statement
3. Validate package name — lowercase, single word, valid Go identifier
4. Execute transformation phases per checklist
5. Verify: `go build ./...` and `go vet ./...` must pass, no remaining `samoa` references
6. Present summary of changes to user
7. Remove this skill directory — it has served its purpose

## Specifications

### Inputs

| Input | Required | Validation | Example |
|-------|----------|------------|---------|
| Package name | Yes | Lowercase, no hyphens, valid Go identifier | `tonga` |
| One-line description | Yes | Single sentence, no period | `Structured logging for Go applications` |
| Purpose statement | Yes | One sentence for MISSION.md Purpose section | `Provide structured, leveled logging with zero-allocation hot paths` |

### MISSION.md Template

The rewritten MISSION.md uses the user's purpose statement and provides blank sections with guidance comments. Do not invent content for "What This Package Contains", "Does NOT Contain", "Success Criteria", or "Non-Goals" — leave these as templates for the user to fill in after building.

### CRITERIA.md Template

The rewritten CRITERIA.md preserves the header and all section headings from the original. Clear all samoa-specific content. Leave blank templates with guidance comments so Armitage has the structure ready when the user populates it.

### README.md Stub

Replace the entire README with the package title, all eight badges (substituted), and the one-line description. Nothing else. The `/review-readme` skill handles full README content.

### Self-Removal

After all transformations and verification pass, delete this skill directory entirely. The `/init` skill is single-use scaffolding.

## Prohibitions

- DO NOT modify agent definitions or other skill definitions
- DO NOT modify PHILOSOPHY.md, STANDING-ORDERS.md, or REVIEW-ORDERS.md
- DO NOT add application code or package-specific logic
- DO NOT run `git commit` or push to remote
- DO NOT populate MISSION.md or CRITERIA.md with guessed content
- DO NOT modify LICENSE, .golangci.yml, .codecov.yml, or CI workflows other than release.yml

## Output

Present a summary after completion:

```text
Package initialized: {name}
Files modified: [list]
Files rewritten: MISSION.md, CRITERIA.md, README.md
Skill removed: .claude/skills/init/

Next steps:
- Review the changes
- Populate MISSION.md with package details
- Populate CRITERIA.md with review criteria
- Run /review-readme when ready to build full README
```
