---
version: alpha
name: Cloud-Init Portal Console
description: Minimal, utilitarian operations UI for generating and monitoring one active cloud-init provisioning configuration.
colors:
  primary: "#111111"
  secondary: "#555555"
  surface: "#FFFFFF"
  surfaceSubtle: "#F5F5F5"
  borderDefault: "#DDDDDD"
  codeBackground: "#111111"
  codeText: "#EEEEEE"
  danger: "#B40000"
  dangerText: "#FFFFFF"
  successBackground: "#E9FFF0"
  successBorder: "#77CC77"
  errorBackground: "#FFE9E9"
  errorBorder: "#EE8888"

typography:
  h1:
    fontFamily: sans-serif
    fontSize: 2em
    fontWeight: 700
    lineHeight: 1.2
  h2:
    fontFamily: sans-serif
    fontSize: 1.5em
    fontWeight: 700
    lineHeight: 1.25
  h3:
    fontFamily: sans-serif
    fontSize: 1.17em
    fontWeight: 700
    lineHeight: 1.3
  body:
    fontFamily: sans-serif
    fontSize: 17px
    fontWeight: 400
    lineHeight: 1.4
  label:
    fontFamily: sans-serif
    fontSize: 1em
    fontWeight: 700
    lineHeight: 1.3
  input:
    fontFamily: sans-serif
    fontSize: 16px
    fontWeight: 400
    lineHeight: 1.2
  button:
    fontFamily: sans-serif
    fontSize: 16px
    fontWeight: 400
    lineHeight: 1.2
  feedbackSuccess:
    fontFamily: sans-serif
    fontSize: 18px
    fontWeight: 700
    lineHeight: 1.3
  codeInline:
    fontFamily: monospace
    fontSize: 0.95em
    fontWeight: 400
    lineHeight: 1.3
  codeBlock:
    fontFamily: monospace
    fontSize: 0.9em
    fontWeight: 400
    lineHeight: 1.4

rounded:
  none: 0px
  sm: 4px
  md: 8px

spacing:
  pageMarginY: 2rem
  pageMarginX: auto
  pageMaxWidth: 980px
  labelTop: 0.6rem
  controlPadding: 0.55rem
  buttonMarginTop: 0.8rem
  buttonPaddingY: 0.6rem
  buttonPaddingX: 1rem
  feedbackPadding: 0.7rem
  feedbackMarginBottom: 0.7rem
  cardPadding: 0.9rem
  cardMarginTop: 1rem
  tableCellPadding: 0.55rem
  rowGap: 0.7rem
  codeBlockPadding: 0.8rem

elevation:
  strategy: border-only
  borderWidth: 1px
  borderColor: "{colors.borderDefault}"

shadows:
  none: none

motion:
  strategy: static
  transitionFast: 0ms
  transitionStandard: 0ms
  autoRefreshInterval: 7000ms

components:
  page:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    typography: "{typography.body}"
    width: "{spacing.pageMaxWidth}"
  card:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    rounded: "{rounded.md}"
    padding: "{spacing.cardPadding}"
  input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    typography: "{typography.input}"
    padding: "{spacing.controlPadding}"
    width: 100%
  select:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    typography: "{typography.input}"
    padding: "{spacing.controlPadding}"
    width: 100%
  button:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    typography: "{typography.button}"
    padding: "{spacing.buttonPaddingY} {spacing.buttonPaddingX}"
  button-danger:
    backgroundColor: "{colors.danger}"
    textColor: "{colors.dangerText}"
    typography: "{typography.button}"
    rounded: "{rounded.none}"
    padding: "{spacing.buttonPaddingY} {spacing.buttonPaddingX}"
  feedback-error:
    backgroundColor: "{colors.errorBackground}"
    textColor: "{colors.primary}"
    padding: "{spacing.feedbackPadding}"
  feedback-success:
    backgroundColor: "{colors.successBackground}"
    textColor: "{colors.primary}"
    typography: "{typography.feedbackSuccess}"
    padding: "{spacing.feedbackPadding}"
  table-status-cell:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    padding: "{spacing.tableCellPadding}"
  table-status-header:
    backgroundColor: "{colors.surfaceSubtle}"
    textColor: "{colors.primary}"
    typography: "{typography.label}"
    width: 220px
  code-block:
    backgroundColor: "{colors.codeBackground}"
    textColor: "{colors.codeText}"
    typography: "{typography.codeBlock}"
    padding: "{spacing.codeBlockPadding}"
---

## Overview

This interface is an operations console: direct, high-density, and practical. It prioritizes immediate task completion over decorative polish. The visual identity is intentionally restrained so field labels, status values, and procedural instructions dominate attention.

The product feels like a trustworthy internal tool: one screen, one primary workflow, minimal ambiguity.

## Colors

The palette is mostly neutral with role-based accents:

- Primary text uses near-black for high legibility.
- Muted secondary text supports explanatory copy without competing with form content.
- Borders are light gray and carry most structural hierarchy.
- Error and success are handled with soft tinted backgrounds and explicit border cues.
- Dangerous actions are rendered with a saturated red background and white text.
- Code/output blocks invert to dark background with light text for strong contrast and easy copy-reference scanning.

## Typography

Typography follows a single-family system stack with semantically different weights and sizes:

- Body copy is slightly enlarged (17px) for readability in long operational sessions.
- Labels are bold to make form scanning quick.
- Inputs and buttons use 16px for clarity and consistent control sizing.
- Success feedback is larger and bold to signal completion.
- Code snippets use monospace to distinguish command syntax and endpoint fragments.

The hierarchy is standard HTML-leaning and intentionally unsurprising.

## Layout

Layout is a centered fixed-max container with stacked cards:

- Page width is capped at 980px.
- Vertical rhythm uses compact rem-based spacing.
- Forms are single-column and full-width controls.
- Horizontal grouping appears only for adjacent action buttons.
- Status data is presented as a key/value table for immediate operational readability.

Spacing choices optimize scan speed and reduce cognitive load in repeated provisioning tasks.

## Elevation & Depth

This system is visually flat. Hierarchy is achieved by borders, section grouping, and heading scale rather than shadows.

Cards and table cells are separated with 1px strokes. The design avoids blur and layered depth effects to keep the interface crisp and deterministic.

## Shapes

Shape language is conservative:

- Cards use medium rounding (8px) to avoid harsh panel edges.
- Most controls remain effectively native in silhouette.
- Danger buttons drop decorative border treatment and rely on color for emphasis.

Overall geometry stays simple and utilitarian.

## Components

Component intent by role:

- Card: primary containment unit for status, instructions, and form workflows.
- Status table: explicit key/value telemetry with highlighted header cells.
- Feedback banners: low-friction inline messaging for success and error states.
- Default buttons: neutral controls for non-destructive actions.
- Danger button: high-salience destructive override (force replace).
- Code block: high-contrast technical output region for command snippets and test instructions.

Interactive behavior is minimal and explicit. The only periodic motion is status refresh polling; there are no animated transitions.

## Do's and Don'ts

Do:

- Preserve the neutral baseline and reserve strong color for semantic states.
- Keep labels bold and controls full-width for fast data entry.
- Maintain border-based separation and flat depth.
- Keep success/error messaging inline and immediately visible near workflow context.

Don't:

- Introduce decorative gradients, heavy shadows, or glassmorphism effects.
- Replace monospace code regions with proportional text.
- Add visual complexity that competes with status telemetry and form completion.
- Use red for non-destructive actions.