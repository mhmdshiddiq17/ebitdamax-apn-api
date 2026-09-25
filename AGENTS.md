# AI Slop Design

- Activate the `ai-slop-design` skill whenever you create or edit files under `resources/js/pages` or `resources/js/components`, or when choosing colors, typography, spacing, radius, shadows, icons, charts, or empty states—and when asked to review, audit, or redesign a screen.
- Skill source: `.opencode/skills/ai-slop-design/SKILL.md` (mirrored to `.kiro/skills` and `.factory/skills`).
- Hard rules: use design tokens from `resources/css/app.css` only (no raw palette colors), keep the red primary (no purple/indigo gradients), reuse `@/components/ui` and `@/lib/formatters`, write UI copy in Bahasa Indonesia, and support light/dark mode plus 360px width.
- Run the skill's "Slop Gate" checklist before reporting frontend work as done.