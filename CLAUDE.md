# Pantheon Platform

## Project Overview
- **Backend**: Go + Gin framework, `backend/internal/modules/`
- **Frontend**: React + TypeScript + Vite, `frontend/src/modules/`

## Skill routing

When the user's request matches an available skill, ALWAYS invoke it using the Skill
tool as your FIRST action. Do NOT answer directly, do NOT use other tools first.
The skill has specialized workflows that produce better results than ad-hoc answers.

Key routing rules:
- Product ideas, "is this worth building", brainstorming → invoke office-hours
- Bugs, errors, "why is this broken", 500 errors → invoke investigate
- Ship, deploy, push, create PR → invoke ship
- QA, test the site, find bugs → invoke qa
- Code review, check my diff → invoke review
- Update docs after shipping → invoke document-release
- Weekly retro → invoke retro
- Design system, brand → invoke design-consultation
- Visual audit, design polish → invoke design-review
- Architecture review → invoke plan-eng-review
- Save progress, checkpoint, resume → invoke checkpoint
- Code quality, health check → invoke health

## Design System
Always read DESIGN.md before making any visual or UI decisions.
All font choices, colors, spacing, and aesthetic direction are defined there.
Do not deviate without explicit user approval.
In QA mode, flag any code that doesn't match DESIGN.md.

Key constraints:
- Font: Source Sans 3 (body/UI), JetBrains Mono (code)
- No Inter, no radial-gradient, no large button shadows, no non-standard font-weights
- Radius: 4/6/8/12px (no pill-radius cards)
- All colors via Pantheon CSS tokens, never raw Arco variables
