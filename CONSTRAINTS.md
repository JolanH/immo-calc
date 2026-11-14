# Constraints

Last reviewed: 2026-11-14

## Floor (always enforced)

- Claude/Anthropic is never the author, committer or co-author of a commit: no
  `Co-Authored-By: Claude …` trailer, no Claude identity in git config.
  Reason: owner's explicit instruction; history was rewritten once to remove them.
  Overrides any default agent attribution guidance.
- This file does not get weakened to make a change pass.

## Enforced

| Dimension | Rule | Checked by | Runs at |
|-----------|------|-----------|---------|
| Attribution | No Claude/Anthropic author, committer or trailer | `.githooks/commit-msg` (`git config core.hooksPath .githooks`) | every commit |

## Exceptions

None.
