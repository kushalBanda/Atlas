---
name: log-it
description: Use when the user says "/log-it", "log this decision", or asks to record an architecture decision, design choice, or trade-off as an ADR. Creates a numbered ADR file under docs/log/ in the current repo.
---

# log-it

## Overview

Records one architecture decision as one ADR file in `docs/log/`. Each file is a permanent record. You write a new file for a changed decision; you never edit an accepted one.

## When to use

- User types `/log-it`.
- User asks to "log this decision," "write an ADR," or "record why we chose X."
- A decision with real trade-offs just got made in conversation (library choice, schema change, API shape, rejected alternative).

Don't use for: routine code changes, bug fixes, or anything with no real alternative considered.

## File location and naming

`docs/log/NNNN-short-title.md`

- `NNNN`: next number, zero-padded to 4 digits. Check existing files in `docs/log/` first; increment the highest number by 1. Start at `0001` if the folder is empty.
- `short-title`: lowercase, hyphenated, 3-6 words. No dates, no ticket numbers.

Example: `docs/log/0001-use-postgres-for-event-store.md`

## Template

```markdown
# NNNN. Title of the decision

Status: Proposed | Accepted | Deprecated | Superseded by NNNN

Date: YYYY-MM-DD

## Context

What problem forces this decision. State the constraint or the trade-off. No filler.

## Decision

The choice, stated as a fact. One or two sentences.

## Consequences

What gets easier. What gets harder. Name the specific cost, not a vague one.

## Alternatives considered

- Option — why rejected, in one line.
- Option — why rejected, in one line.
```

Omit "Alternatives considered" only if none were seriously weighed.

## Writing rules for the content

- Write like people talk. No corporate jargon.
- State the decision directly. Skip "I think," "maybe," "could."
- Active voice.
- Say what something is, not what it isn't.
- Cut banned words on sight: very, really, just, basically, actually, leverage, utilize, robust, seamless, implement, facilitate, best practices, game-changing, innovative, cutting-edge, disrupt.
- No em dashes. Use a comma, a semicolon, or a new sentence.
- No hedge words: might, perhaps, potentially, arguably — unless the uncertainty is real and worth naming.
- Sentence case for headings, not title case.
- One blank line between sections. No stacked blank lines.
- Numbers and facts over adjectives. "Cuts query time from 400ms to 40ms," not "much faster."

## Steps

1. Read `docs/log/` to find the highest existing ADR number.
2. Pull the decision, context, and trade-offs from the conversation. Ask the user only if a required field is missing.
3. Write the file using the template above, following the writing rules.
4. Confirm the file path back to the user.

## Common mistakes

- Editing an old ADR instead of writing a new one that supersedes it. Set the old one's status to `Superseded by NNNN` and link the new number.
- Vague consequences ("better performance") instead of a concrete cost or gain.
- Listing alternatives with no reason they were rejected.
