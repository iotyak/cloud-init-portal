---
version: alpha
name: Cloud-Init Provisioning Console
description: Minimal operations UI focused on creating one provisioning config and tracking its generated/consumed lifecycle.
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
  errorBackground: "#FFE9E9"

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
  pageMaxWidth: 980px
  labelTop: 0.6rem
  controlPadding: 0.55rem
  buttonMarginTop: 0.8rem
  buttonPadding: 0.6rem
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
  config-select:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    typography: "{typography.input}"
    padding: "{spacing.controlPadding}"
    width: 100%
  config-input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    typography: "{typography.input}"
    padding: "{spacing.controlPadding}"
    width: 100%
  button-primary:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    typography: "{typography.button}"
    padding: "{spacing.buttonPadding}"
  button-danger:
    backgroundColor: "{colors.danger}"
    textColor: "{colors.dangerText}"
    typography: "{typography.button}"
    rounded: "{rounded.none}"
    padding: "{spacing.buttonPadding}"
  feedback-error:
    backgroundColor: "{colors.errorBackground}"
    textColor: "{colors.primary}"
    padding: "{spacing.feedbackPadding}"
  feedback-success:
    backgroundColor: "{colors.successBackground}"
    textColor: "{colors.primary}"
    typography: "{typography.feedbackSuccess}"
    padding: "{spacing.feedbackPadding}"
  status-table-header:
    backgroundColor: "{colors.surfaceSubtle}"
    textColor: "{colors.primary}"
    typography: "{typography.label}"
    width: 220px
  status-table-cell:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    padding: "{spacing.tableCellPadding}"
  generated-details-code:
    backgroundColor: "{colors.codeBackground}"
    textColor: "{colors.codeText}"
    typography: "{typography.codeBlock}"
    padding: "{spacing.codeBlockPadding}"
---

## Overview

This UI is a strict provisioning console with one purpose: capture the minimum configuration required to generate a cloud-init configuration, then show what was generated and whether it has been consumed.

The interface must remain narrowly scoped, operational, and deterministic. The design should discourage feature creep and keep the workflow obvious.

## Colors

The visual palette is neutral-first:

- Near-black primary text for readability.
- Muted secondary text for helper copy.
- Light gray borders and subtle table headers for structure.
- Soft green for success confirmation.
- Soft red for validation or system errors.
- Strong red only for destructive/override actions.
- Inverted dark code panels for generated command/snippet details.

## Typography

Typography uses a simple system sans stack and clear weight hierarchy:

- Body text at 17px for comfortable scanning during ops workflows.
- Bold labels for form fields and status keys.
- 16px controls for consistent input ergonomics.
- Larger bold success feedback to signal completed generation.
- Monospace for generated technical details and endpoint snippets.

## Layout

The screen is a centered, fixed-width console with stacked cards.

The configuration form must present exactly these user-configurable inputs:

1. Cloud-init template (select)
2. Box type (hardware vendor/model, select)
3. Hostname
4. IP address
5. CIDR
6. Gateway
7. DNS servers

No additional editable configuration controls should be introduced in this form.

After generation, layout must prioritize a status/details area that clearly surfaces:

- generated configuration identity (hostname, template, box type, network values)
- generated timestamp
- current lifecycle state, including whether the generated configuration has been consumed

## Elevation & Depth

Depth is intentionally flat. Hierarchy is communicated through card borders, heading levels, and table structure rather than shadow-heavy layering.

## Shapes

Shape treatment is conservative and utility-first:

- Cards use 8px rounding.
- Inputs/selects follow native rectangular geometry.
- Destructive action button remains visually distinct via red fill.

## Components

Key components are constrained to the provisioning workflow:

- Configuration form card with the seven approved fields.
- Generate action button for creating the config.
- Status/details card with key/value telemetry.
- Inline success/error feedback banners.
- Generated detail blocks for copyable output.
- Consumed-state visibility in the status area (must be explicit, never implicit).

Component additions outside this workflow should be treated as out-of-scope unless they directly improve generation clarity or consumed-state visibility.

## Do's and Don'ts

Do:

- Keep the form limited to template, box type, hostname, IP, CIDR, gateway, and DNS.
- Keep generated details and consumed status visible without extra navigation.
- Preserve high-contrast text and border-based structure.
- Keep feedback inline and close to the relevant workflow step.

Don't:

- Add optional configuration knobs unrelated to provisioning generation.
- Hide consumed state behind ambiguous wording or deep navigation.
- Introduce decorative visuals that reduce scan speed.
- Use destructive color treatment for non-destructive actions.
