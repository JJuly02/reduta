# ADR-0008 - Frontend: React + TanStack Query, plain CSS design tokens

**Status:** accepted for M6; revisit if the admin table needs virtualization

Spec section 3/8 lists Tailwind 4 + shadcn/ui, TanStack Table/Virtual, cmdk and
dnd-kit. For M6 the SPA uses React 19 + Vite + TypeScript + React Router 7 +
**TanStack Query** for server cache, with a **hand-written CSS** design system
that follows the spec's visual tokens (section 8.1: oklch surfaces, single cool
accent, dark-first, sharp edges, JetBrains Mono for flags/hosts). This keeps the
build dependency-light and reliable while matching the intended look.

Deferred (documented deviation, spec section 0.5): Tailwind/shadcn, TanStack
Table + Virtual (the admin challenge table is a plain table with client-side
selection for now; server-side pagination + virtualization is the M6 follow-up
for 200+ rows), cmdk command palette, dnd-kit block drag and drop. The realtime
scoreboard and challenge board already update live over the WebSocket from M5.
