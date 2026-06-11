---
name: frontend-visual-regression-testing
description: Test frontend UI flows with screenshot-driven verification, continuous navigation, frame/timing probes, and visual irregularity analysis. Use when Codex needs to verify reader navigation, chapter/page transitions, localhost UI behavior, brief flashes, blank states, layout shifts, clipped text, incorrect selected states, or any frontend change that needs visual proof before reporting success.
---

# Frontend Visual Regression Testing

## Overview

Use this skill to verify UI behavior by actually opening the app, moving through the full affected workflow, capturing screenshots, and sampling fast transition states that may be too quick for a human to capture.

## Workflow

1. Identify the exact user flow and its boundaries.
   - For navigation bugs, traverse the whole flow, not only one local segment.
   - For reader chapter navigation, test first chapter to last chapter, page boundaries, TOC selection, previous/next buttons, and keyboard navigation when relevant.
   - Include the user-reported edge case explicitly, such as a 0.1s flash during chapter changes.

2. Start the smallest useful app surface.
   - Prefer the repo command in `AGENTS.md`.
   - Use `npm run dev` from `frontend` when a Vite-only harness is enough.
   - Use `wails dev` when native/Wails behavior matters.
   - Record any missing prerequisite instead of changing the toolchain.

3. Open the app in the available browser testing tool.
   - Prefer the in-app Browser when it works for the target.
   - If localhost is blocked or the in-app browser cannot render the harness, use a local browser automation fallback and report that fallback.
   - Keep browser work in the background unless the user asks to watch it.

4. Capture screenshots continuously.
   - Capture initial state, before each important interaction, immediately after interaction, after transition delay, and final settled state.
   - Save screenshots under a temp directory, using names that describe the state, for example `01-before-boundary.png`, `02-after-boundary-immediate.png`, `03-after-boundary-50ms.png`, `04-after-boundary-settled.png`.
   - Analyze every screenshot before continuing.

5. Probe fast visual glitches with DOM/frame sampling.
   - After a suspicious click, sample state immediately and again after short delays such as 16ms, 50ms, 100ms, 200ms, and 400ms.
   - Capture computed visibility, class names, selected/active state, page/chapter labels, progress labels, transforms/offsets, and text length when those values explain the visual.
   - For hidden transition states, verify both class intent and computed style, for example `is-hidden` plus `visibility: hidden`.

6. Look for irregular behavior.
   - Blank or unexpectedly full-text screens.
   - Wrong chapter/page/progress label.
   - Incorrect active TOC/sidebar state.
   - Buttons enabled or disabled at the wrong time.
   - Stuck loading or hidden content.
   - Horizontal or vertical overflow.
   - Layout shift, clipped text, overlap, or unreadable contrast.
   - Console/runtime errors that correlate with the visual flow.

7. Fix in scope, then retest the same flow.
   - After any fix, rerun the failing interaction and the broader navigation path around it.
   - For chapter navigation, retest both within-chapter paging and chapter-boundary crossing.
   - Do not call the issue fixed until the after-fix screenshot/frame evidence covers the original failure.

8. Clean up temporary scaffolding.
   - Remove temporary harness files from the repo.
   - Stop dev servers started for the test.
   - Remove temporary automation packages when safe.
   - Check `git status --short --untracked-files=all` before the final report.

## Reporting

Lead with whether the flow passed or failed. Include:

- What route/app URL was tested.
- What interactions were performed, especially full traversal counts.
- Screenshot paths for key before/immediate/settled/final states.
- Any frame-sampling evidence for quick flashes.
- Commands run, such as `go test ./...` and `npm run build`.
- Known warnings or gaps.
- Any browser fallback used because the preferred browser could not render the target.

Keep the report concrete. For a 0.1s transition bug, say exactly what was visible immediately after the click and when the settled state became visible.
