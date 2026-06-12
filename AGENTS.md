# AGENTS.md

Guidance for coding agents working in this repository.

## Project

This project is a desktop EPUB reader built with Wails and the React template.

The intended product is a local-first desktop app for importing, managing, reading, and studying EPUB books. The UI direction should be inspired by Readest, especially its library, reader, settings, table-of-contents, and translation/lookup workflows.

Public Readest references may be used for direction:

- https://readest.com/
- https://github.com/readest/readest
- https://readest.com/docs/customization

Do not assume exact visual cloning is required unless the user provides screenshots or explicit UI requirements.

## Required Product Functions

The app must support:

- Importing `.epub` files.
- Copying imported EPUB files into the app's managed local data folder.
- Managing imported books in a local library.
- Reading books.
- Changing reader background color.
- Changing reader font size.
- Showing and navigating the table of contents.
- Selecting a word or text while reading and opening translation/study help.
- Opening ChatGPT or Gemini in a popup/window for AI translation and study help.
- Opening Google Translate externally without paid Google Cloud Translation API usage.
- Managing prompts per book.
- Creating book-aware AI prompts that include enough context from the book to improve translation quality.
- Supporting language-learning output from AI, including new vocabulary, meaning, and grammar explanation for translated text.

The app should not embed a paid or API-backed AI provider unless the user explicitly changes this requirement. Prefer opening user-facing ChatGPT/Gemini/Google Translate surfaces and preparing/copying useful prompts or text for them.

## Example AI Study Use Case

For a book named `Cradle`, the user may select text and ask AI to translate it. The AI prompt should be aware that the content comes from `Cradle` and should provide:

- A natural translation with proper book context.
- Important vocabulary from the selected text.
- Short explanations of useful grammar patterns.
- Any nuance needed to understand the passage.

Per-book prompt management should allow the user to customize this behavior for each book.

## Tech Stack

Current scaffold:

- Wails v2 desktop app.
- Go backend.
- React frontend.
- Vite frontend build.
- npm package manager for the frontend.

Key files:

- `main.go` and `app.go`: Wails application entry/backend binding.
- `go.mod`: Go module and Wails dependency.
- `wails.json`: Wails project configuration.
- `frontend/package.json`: frontend scripts and dependencies.
- `frontend/src`: React application code.

## Commands

Use these commands from the repository root unless noted otherwise:

- `wails dev`: run the desktop app in live development mode.
- `wails build`: build a redistributable desktop app.
- `wails build -m -skipbindings -skipembedcreate`: build after `go vet` and binding generation are already current if Wails static analysis walks `frontend/node_modules`.
- `go test ./...`: run Go tests.
- `go fmt ./...`: format Go code.
- `go vet ./...`: run Go static checks when useful.
- `npm install`: install frontend dependencies from `frontend`.
- `npm run build`: build the frontend from `frontend`.
- `npm run dev`: run the frontend dev server from `frontend`.

If commands fail because required tools are missing, report the missing prerequisite and do not replace the project toolchain without user approval.

## Go Conventions

Follow standard Go conventions:

- Run `gofmt` on Go changes.
- Use idiomatic package names: short, lowercase, no underscores.
- Keep errors explicit and return them instead of hiding failures.
- Prefer small interfaces only where they are consumed.
- Keep exported names documented when they are part of a public package surface.
- Avoid unnecessary abstraction and JavaScript-style patterns in Go code.
- Use standard library functionality before adding dependencies.
- Keep filesystem paths platform-aware. This is a desktop app and must work on Windows first unless the user says otherwise.

## Frontend Conventions

Follow the existing React/Vite structure unless there is a clear reason to change it.

Reader UI should be practical and quiet:

- Library view for imported books.
- Reader view focused on text.
- TOC/sidebar access.
- Reader settings for background color and font size.
- Translation/study action available from selected text.
- Prompt management available per book.

Avoid building a marketing landing page inside the app. The first useful screen should be the library or reader workflow.

## Local Data Expectations

Imported EPUB files should be copied into the app's managed data directory rather than referenced only by their original path.

Book metadata should track at least:

- Stable book ID.
- Title.
- Author when available.
- Imported file path inside app data.
- Original filename.
- Import timestamp.
- Reading progress.
- Per-book prompt configuration.

Do not store secrets or private user content in logs.

## Planning

Use spec-driven development for substantial product work.

Before implementation, make sure the intended behavior is written down in a Linear ticket, linked project note, or local plan. For small fixes, a concise plan in the conversation is enough. Larger specs should include:

- Problem statement.
- User workflow.
- Functional requirements.
- Non-goals.
- Data model or API changes, if any.
- UI states and interactions.
- Acceptance criteria.
- Verification plan.

Implementation should follow the accepted spec. If the implementation needs to diverge, update the ticket/spec with the reason before continuing.

## Local Agent Skills

The repository may contain reusable agent skills under `.agent/`.

For non-trivial work, inspect and use applicable `.agent/*/SKILL.md` guidance before planning or implementation. Announce which local skills are being used and why.

Useful local skills may include:

- `spec-driven-development` for Linear/spec-first work.
- `source-driven-development` for grounding implementation in current code and authoritative sources.
- `incremental-implementation` for small, safe implementation steps.
- `code-review-and-quality` for review and verification passes.
- `frontend-ui-engineering` for UI tickets.
- `frontend-visual-regression-testing` for screenshot-driven UI flow testing, reader navigation checks, and fast transition/frame-sampling bugs.
- `git-workflow-and-versioning` for commits and branch hygiene.

Do not force a skill when it does not apply. If a relevant skill is missing, unreadable, or conflicts with this `AGENTS.md`, state the issue and follow `AGENTS.md`.

## Collaboration Rules

- Prefer small, reviewable changes.
- Preserve user changes and never revert unrelated work.
- Ask before adding major dependencies.
- Keep generated files and lockfiles consistent with the command that produced them.
- Record important product decisions in the relevant ticket, note, or final response.
- If Readest UI behavior is used as a reference, mention the specific public page or user-provided screenshot that informed the decision.
