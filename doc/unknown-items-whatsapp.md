# Unknown items: pictures over WhatsApp (plan, not built)

A second way to picture unknown items, next to the QR phone page. A branch
setting chooses which one the till uses. With WhatsApp, salespersons send each
unknown item's picture into a WhatsApp chat (usually a group) picked in kashi,
captioned with the code the till shows (`u1`, `u2`, …). The till holds the
receipt until every unit has a picture, exactly as with QR.

## What already exists

- kashi: one slot per unit, units can share one picture, and a line's units
  are grouped by `line_ref` (one picture fills the whole line). The till's
  long poll (`branchUnknownSessionStatus`) and the "every unit has a picture or
  a manager's pass" check stay as they are.
- Bridge (bnc `whatsapp-bridge`, branch `bridge-multi-session`): it already
  sees every incoming message (`messages.upsert`), downloads media
  (`saveMedia`), passes chats on to kashi (`logChat`, which skips groups
  today), and lists the number's groups (`GET /group/fetchAllGroups`).

## How it works

- **Setting**: branch settings → unknown item pictures: `QR` or `WhatsApp
  chat`, plus the chat, picked from the branch number's groups and chats.
- **Bridge**: images (and text, for `u1x2` sent on its own) from a chat kashi
  has selected are POSTed to kashi with the message id, chat id, sender,
  caption, WhatsApp's send time and the downloaded image. It must also pass on
  the backlog delivered after a reconnect (`type === 'append'`), not only live
  `notify` messages.
- **Caption**: `u<n>` fills every unit of line n; `u<n>x<k>` fills k units.
  Parse loosely: any case, spaces (`u 1`, `u1 x2`), Arabic digits, `x`/`×`/`*`.
  More pictures or a bigger `k` than units: fill what's there, flag the rest.
- **No caption**: an uncaptioned picture from the same sender within ~30s of
  a captioned one counts for the same line (WhatsApp captions only the first
  of several selected photos). Must be per sender, so two people sending at
  once don't mix.
- **Sender**: not used for matching; the chat is the trust boundary (who can
  post is who is in the group). Stored with the picture, for the Unknown items
  page.

## Numbering

- Codes are unique per **chat**, across every waiting checkout of every
  branch using that chat. CT1 gets `u1, u2`, CT2 starting at the same moment
  gets `u3`. Handed out in one transaction with a unique index on (chat,
  number) among waiting checkouts, so two tills can't both get `u1`.
- Lowest free number first; numbers free up when the sale lands or the cart is
  cleared (the till releases them, not only expiry).
- A freed number isn't reused for a couple of minutes, and only pictures whose
  WhatsApp send time is after the checkout started count, so a late picture
  for an old sale can't land on a new one.

## Several branches in one group

- Works with the per-chat numbering above.
- If two branches' kashi numbers are both in the group, each receives the
  same message: drop duplicates by WhatsApp message id, and let only one
  number (the chat's first branch, or a chosen one) react and reply.
- Default advice: one group per branch (small numbers, quiet chat, a typo
  can't land in another branch).

## Making it easy for salespersons

- kashi reacts ✅ on an accepted picture and ❓ on one it can't place, and
  replies "u2 still needs 1 picture". Staff group, so no restriction risk.
- When a chat is selected, kashi sends and pins an instructions message, e.g.
  "Picture each unknown item, caption it with the code on the till screen:
  u1, u2… Several of the same item: u1x2, or send 2 pictures captioned u1.
  Wait for ✅." (plus Arabic).
- The till shows the codes large, ticking each as it lands.
- Manager pass and "Ask a manager" work unchanged.

## Reliability

- Speed: ~2–4s from send to the till (WhatsApp delivery ~1s, media download
  under 1–2s, till long poll 250ms).
- The bridge is unofficial WhatsApp Web and can drop. While the branch's
  session is down, the till says so and offers QR at once; that fallback is
  why QR stays.

## Work, per repo (each on its own `unknown-whatsapp` branch)

- bridge: pass on media/text from selected chats (live and backlog), react,
  reply, pin.
- kashi: setting + chat picker, inbound endpoint, caption parsing, per-chat
  numbering, dedupe, replies, sender on the picture.
- kashi-pos: show the codes on the waiting screen, release numbers when the
  cart clears, fall back to QR when WhatsApp is down.
