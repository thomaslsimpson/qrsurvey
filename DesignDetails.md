# Handoff: QR Survey Form (Uptown Mall)

## Overview
This is the interactive **feedback form** described in the `qrsurvey` README: a customer scans a QR poster
at a business (here, a mall), lands on a short mobile survey about their visit, and is entered into a prize
draw on completion. The form is intentionally delightful — light-up controls, satisfying motion, and a
progress bar that counts down the questions remaining until the prize entry.

Flow (per the repo README):
1. **Welcome** — inviting banner that mentions the prize draw.
2. **Questions 1–4** — one question per screen, each enjoyable to answer, each showing "X questions until your entry".
3. **Contest entry** — contact details (name, email, phone).
4. **Thanks** — confirmation that they're entered.

The poster design is explicitly out of scope (the repo README defers it to "another time").

## About the Design Files
The files in `prototypes/` are **design references created as HTML** (specifically "Design Component"
`.dc.html` files — a small React-based streaming component format). They are prototypes demonstrating the
intended **look, copy, and behavior**, not production code to ship verbatim.

The task for the implementing agent is to **recreate these designs in the target codebase's environment**
using its established patterns and libraries. The `qrsurvey` repo currently contains only a README, so there
is **no existing app environment** — the implementer should choose an appropriate stack. A suggested stack
given the requirements (mobile web, reached via QR link, form + DB persistence, client dashboard later):
- **Frontend:** React (Vite or Next.js), or plain HTML/CSS/JS if kept minimal. Framer Motion (or CSS
  transitions) for the animations.
- **Backend/DB:** any simple API + relational store (Postgres/SQLite) or a BaaS (Supabase/Firebase) to
  store responses keyed by business/campaign and to manage prize-draw entries.
- The `.dc.html` prototypes are readable as plain HTML+JS: the markup is in the `<x-dc>` template and the
  behavior is in the `class Component extends DCLogic` block. Lift the exact values (hex, spacing, timing,
  copy) from them.

## Fidelity
**High-fidelity.** Final colors, typography, spacing, copy, and interactions are all present. Recreate the UI
closely, adapting only to the target framework's idioms. Four visual directions are provided — the client
should pick ONE to carry forward; they are not meant to coexist.

## Two versions in this bundle
- **`Uptown Mall Survey.dc.html`** (v1) — answer selection via a **vertical stack of 5 tappable option
  buttons** that light up on select and auto-advance.
- **`Uptown Mall Survey v2.dc.html`** (v2, latest) — answer selection via a **horizontal slider** (worst→best)
  with no default, five notches, star clusters (1–5 ★) above each notch that glow on interaction, and a
  **Next** button that stays disabled until a rating is chosen. **v2 is the current preferred interaction.**

Everything else (welcome, contest, thanks, progress bar, transitions, palettes) is identical between the two.

## The four design directions
Each is a full, independent flow rendered inside a mobile phone frame (390-style portrait; drawn at 336×736
screen inside a 362×762 bezel). Pick one.

| id (v1 / v2) | Name | Vibe | Fonts (heading / body) | Key colors |
|---|---|---|---|---|
| 1a / 2a | **Sunburst** | Playful & bright | Fredoka / Nunito | coral `#FF5A5F`, tangerine `#FF9F1C`, cream bg `#FFF4E6`→`#FFE7E4` |
| 1b / 2b | **Cocoa & Amber** | Warm & friendly | DM Serif Display / DM Sans | terracotta `#B5563A`, amber `#DA8C3D`, cream bg `#F7EDE2` |
| 1c / 2c | **After Dark** | Sleek & modern (dark) | Space Grotesk / Sora | gold `#FFB627`, ember `#FF6B35`, near-black bg `#120E0B`/`#241A15` |
| 1d / 2d | **Prize Carnival** | Bold & exciting | Righteous / Nunito | yellow `#FFC93C`, orange `#FF7A45`, magenta→red bg `#6D1B4F`→`#9E2A5A`→`#C2412E` |

## Screens / Views

### 1. Welcome
- **Purpose:** Invite the customer in and surface the prize hook.
- **Layout:** Vertical flex column, 24px horizontal padding. Top: a ~206px-tall hero card (rounded 20–26px)
  with a "PRIZE DRAW" / "WIN BIG" badge and a short headline. Below: an h1 greeting, a one-line subtitle,
  a flexible spacer, then a full-width primary CTA pinned to the bottom.
- **Copy:** Headline references "Uptown Mall"; subtitle: "Answer 4 quick questions and you're automatically
  entered to win. Takes about 30 seconds." CTA per theme: "Let's go →" / "Begin →" / "Start →" / "LET'S PLAY →".
- **No header/progress bar on this screen.**

### 2. Question (×4)
- **Purpose:** Collect one rating per screen.
- **Header (persistent on question + contest screens):** back arrow (‹) in a 34px circle, a 10px-tall
  rounded progress track with a gradient fill, and a countdown line: "N questions until your entry"
  (singular "1 question…" at the end).
- **v1 body:** uppercase kicker (question category), h2 prompt (24–27px), then a 12px-gap vertical stack of
  5 full-width option buttons. Each button: left-aligned label + right-side marker circle. Unselected =
  white/tinted card with subtle border and outline circle. Selected = theme gradient/fill, white or dark
  label, glowing shadow, `scale(1.03)`, filled ✓ circle. Selecting auto-advances after ~420ms.
- **v2 body:** uppercase kicker, h2 prompt, a live **current-selection label** (theme accent color, ~22px)
  that reads "Slide to rate →" until a choice is made. Then the slider block:
  - **Star row:** 5 equal columns; column *i* holds *i* star glyphs (★) — so 1,2,3,4,5 stars left→right,
    signalling worst→best hierarchy. Stars are dim (`starOff`) until their notch is selected, then switch to
    `starOn` color with a `drop-shadow(...)` **glow** and `scale(1.15)`.
  - **Track:** a 6px rounded base line spanning 10%→90% of the width; a gradient **fill** from 10% to the
    selected notch; 5 notch dots at 10/30/50/70/90%; a 26px circular **thumb** (hidden until first selection)
    that snaps to the selected notch.
  - **End labels:** the extreme option labels under the ends (e.g. "Poor" … "Loved it").
  - **Next button:** full-width, "Next →". **Disabled** (grey `nextOff`/`nextOffText`, `cursor:not-allowed`)
    until a rating exists; enabled = theme `nextOn` fill.
  - Interaction: pointer-drag on the track (pointerdown/move/up with pointer capture) maps X to the nearest
    of 5 notches; tapping a star column or a dot also selects. No auto-advance by default (there is an
    optional `autoAdvanceOnPick` toggle).

  **Question content (worst → best order for the slider):**
  - Q1 · *Overall experience* — "How was your visit to Uptown Mall today?" — Poor / Not great / It was okay / Pretty good / Loved it
  - Q2 · *Recommend to a friend* — "How likely are you to recommend us to a friend or family?" — Very unlikely / Unlikely / Neutral / Likely / Very likely
  - Q3 · *Ease of visit* — "How easy was it to do what you came here for?" — Very difficult / Difficult / So-so / Easy / Very easy
  - Q4 · *Come back soon* — "How likely are you to visit again this month?" — No / Probably not / Maybe / Probably / Definitely

  (These map to the client's requested questions: overall satisfaction, likelihood to recommend, ease of
  accomplishing their goal, likelihood to return within a month.)

### 3. Contest entry
- **Purpose:** Capture contact info for the prize draw.
- **Layout:** kicker "Last step", h2, one-line reassurance ("Only used to contact the winner"), then 3
  stacked text inputs (Full name / Email address / Mobile number), spacer, full-width submit CTA.
- **Inputs:** full-width, ~15–16px padding, rounded 12–16px, theme border; on focus the border switches to
  the theme primary (`style-focus` in the prototype).
- Header + progress bar still shown (bar reads ~100%).

### 4. Thanks
- **Purpose:** Confirm entry and offer a restart.
- **Layout:** centered column — a 96px circular badge (theme fill) with a ✓ that plays a `pop` keyframe
  animation on mount, an h1 ("You're entered!" / "You're in the draw!"), a reassurance line about the
  end-of-month draw, and a subtle "↺ Take it again / Play again" restart link.

## Interactions & Behavior
- **Step machine:** step index `i` 0–6 → 0 welcome, 1–4 questions, 5 contest, 6 thanks. `back` decrements,
  CTA increments, `restart` resets to 0 with cleared answers.
- **Progress bar:** width = `min(100, round(i/5*100))%`, animated with `width .5s cubic-bezier(.22,1,.36,1)`.
- **Countdown copy:** `remaining = 5 - i` while on a question step.
- **Slide transition between steps (v2 + v1):** the content wrapper starts at `opacity:0, translateX(30px)`
  with no transition, then on the next frame animates to `opacity:1, translateX(0)` via
  `opacity .38s ease, transform .42s cubic-bezier(.22,1,.36,1)`. Respect a **reduced-motion** flag (both
  prototypes expose `reduceMotion`, which disables this).
- **v1 select:** records the answer, then auto-advances after ~420ms so the light-up is seen first.
- **v2 select:** records the answer only; the user taps **Next** to advance. Next is inert until an answer
  exists. Optional `autoAdvanceOnPick` advances ~550ms after a pick.
- **Star glow:** selected notch's stars gain the theme `starGlow` drop-shadow + scale; transitions
  `color .2s, filter .25s, transform .25s`.
- **Button/CTA hover:** `translateY(-2px)` lift (`style-hover`).
- **Thanks badge:** `@keyframes pop` — `scale(.4)→1.12→1` over .5s `cubic-bezier(.34,1.56,.64,1)`.

## State Management
- Per-flow state: `{ i: <stepIndex>, ans: { q1..q4: <notchIndex 1–5 (v2) | label (v1)> }, phase: 'start'|'end' }`.
- `phase` drives the slide-in animation (transient; flips `start`→`end` on the next tick after navigation).
- v2 stores the answer as the **1–5 notch index**; the human label is `opts[index-1]`.
- Data to persist for the real product (not in the prototype): a response record per submission with
  `{ businessId/campaignId, timestamp, q1..q4 ratings, contact {name,email,phone}, consent }`, plus a
  prize-draw entry. The prototype does not call any backend — wire the contest-submit CTA to your API.
- **Note on the prototype file:** it renders four independent flows side-by-side on a design canvas so the
  directions can be compared; the real product ships ONE flow full-screen.

## Design Tokens

### Sunburst (2a)
- primary `#FF5A5F`, accent `#FF9F1C`, ink `#3A2E2A`, muted `#937F74`, hint `#B7A597`
- screen bg `linear-gradient(170deg,#FFF4E6,#FFE7E4)`; card border `#F2DECF`; progress track `#F3D9CE`
- slider: starOff `#E7D3C6`, starOn `#FF9F1C`, glow `drop-shadow(0 0 6px rgba(255,159,28,.8))`;
  fill/dotOn `linear-gradient(120deg,#FF5A5F,#FF9F1C)`; thumb white w/ 3px `#FF5A5F` border
- nextOff `#EFE0D6` / `#C3B2A6`; nextOn coral→amber gradient, white text

### Cocoa & Amber (2b)
- primary `#B5563A`, accent `#DA8C3D`, ink `#3A2418`, muted `#8A7364`/`#A99686`
- screen bg `#F7EDE2`; input border `#E1CDBB`; progress track `#E7D6C6`
- slider: starOff `#E1CDBB`, starOn `#DA8C3D`; fill `linear-gradient(90deg,#B5563A,#DA8C3D)`; dotOn `#B5563A`
- nextOff `#EADBCB` / `#B6A493`; nextOn `#B5563A`, white text

### After Dark (2c)
- gold `#FFB627`, ember `#FF6B35`, ink `#F5EDE6`, muted `#A99C90`/`#8E8073`
- screen bg `radial-gradient(120% 85% at 50% -10%,#241A15,#120E0B 62%)`; card `#1D1611`/`#241C17`; borders `#3A2E27`; track `#2A211C`
- slider: starOff `#3A2E27`, starOn `#FFB627`, glow `drop-shadow(0 0 7px rgba(255,182,39,.9))`; fill/dotOn gold→ember; thumb `#1A1410` w/ gold border + glow
- nextOff `#241C17` / `#6A5C50`; nextOn gold→ember, dark text `#1A1410`

### Prize Carnival (2d)
- yellow `#FFC93C`, orange `#FF7A45`, magenta `#6D1B4F`, wine `#9E2A5A`, red `#C2412E`
- screen bg `linear-gradient(165deg,#6D1B4F,#9E2A5A 55%,#C2412E)`; white text; translucent white borders/inputs `rgba(255,255,255,.14–.4)`; track `rgba(255,255,255,.22)`
- slider: starOff `rgba(255,255,255,.35)`, starOn `#FFC93C`, glow `drop-shadow(0 0 7px rgba(255,201,60,.9))`; dotOn/fill yellow→orange; thumb `#6D1B4F` w/ yellow border
- nextOff `rgba(255,255,255,.18)` / `rgba(255,255,255,.55)`; nextOn `#FFC93C`, magenta text `#6D1B4F`

### Shared scale
- Phone: bezel 362×762 radius 46, padding 13; screen 336×736 radius 34.
- Spacing: 24px screen padding; 12px option gaps; 6px progress/slider bar height; 96px thanks badge; 34px header back button.
- Radii: buttons/cards 12–26px (theme-dependent); pills 99px (Carnival CTA).
- Type sizes: h1 ~28–30px, h2 (prompt) 24–27px, body 16px, kicker 12px uppercase +1.5 letter-spacing, status bar 13px.
- Fonts (Google): Fredoka, Nunito, DM Serif Display, DM Sans, Space Grotesk, Sora, Righteous.

## Assets
- **No image assets.** All visuals are CSS (gradients, circles, dashed "ticket" borders). Star ratings use
  the `★` Unicode glyph. The status bar battery/notch are simple CSS shapes. If the real product wants photos
  (e.g. a hero image on the welcome card), add an image slot there.
- **Fonts** load from Google Fonts.

## Files
- `prototypes/Uptown Mall Survey.dc.html` — v1 (button-select flow), directions 1a–1d.
- `prototypes/Uptown Mall Survey v2.dc.html` — v2 (slider flow), directions 2a–2d. **Preferred.**

Open either file directly in a browser to interact with all four directions. Within each `.dc.html`:
the `<x-dc>…</x-dc>` block is the markup; the `class Component extends DCLogic { … }` block is the logic
(step machine, slider math, style builders). Tweakable props are declared in the trailing `data-props` JSON.
