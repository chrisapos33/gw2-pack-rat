# GW2 Companion Tool — v1 Spec

## Problem

GW2 players accumulate inventory, bank, and material storage they can't quickly evaluate: is this item needed for a recipe, a collection/legendary, or is it just clutter worth selling or salvaging? This was the single most repeated pain point across direct community research (r/Guildwars2 thread, ~103 days old, "what tool do you wish existed"), showing up independently as: material tooltips tied to goals, "what's safe to delete," "scans my inventory and tells me what's worth keeping," missing-mats-for-crafting lookups, and hover-to-see-value requests. No existing free tool (GW2Efficiency, BlishHUD, gw2stacks, gw2pathfinder) solves this end-to-end — they cover pieces (prices, pathing, wiki lookups) but not "look at my inventory and tell me what to do with each stack."

## v1 Core Loop

1. User signs up and connects a GW2 API key scoped to only the permissions the tool needs (account, inventories, characters, unlocks — not full access). Key is validated against `/v2/tokeninfo` before being stored (encrypted at rest).
2. On login/session start, the account is automatically scanned — no manual "scan" button. Scan runs if the last snapshot is older than a short staleness threshold (a few minutes); otherwise the cached snapshot is served. GW2 inventories don't change without the player actively playing, so aggressive caching is fine and keeps API usage low.
3. Shared inventory slots, every character's bags, bank, and material storage are pulled and enriched: each item is tagged with any known use (recipe ingredient, part of a curated collection/legendary) or marked as having no known use.
   - **Note:** `/v2/account/inventory` only returns the account-wide *shared* inventory slots, not character bag contents. Full coverage requires also fetching the account's character list (`/v2/characters`) and calling `/v2/characters/:id/inventory` per character (needs the `characters` scope, already in the permission list below). Missing this step would silently undercount most players' actual clutter, since character bags are where the bulk of it lives.
4. Results view shows items grouped by location (shared inventory / character bags / bank / materials), each with its tag(s) and a recommended action: keep, sell, salvage, or unclear.
   - **Recommended action must branch on the item's `binding` field.** Account-bound and character-bound (soulbound) items cannot be listed on the trading post — for those, "sell" is never a valid recommendation; the choices are salvage, use, or vendor. Only unbound items can be recommended for TP sale. Getting this wrong means telling a player to do something the game won't let them do.
   - When an item has many known uses (e.g. a common material matching 100+ recipes), show a count ("used in 174 recipes") rather than listing every match — v1 doesn't need to enumerate all of them, just signal that the item is useful.

That's the entire v1 product surface. One core screen, one question answered per item.

## Explicitly Out of Scope for v1

Cutting these was a deliberate decision, not an oversight — do not reintroduce without a conscious rescoping conversation:

- Pinned/personalized goals (showing only items relevant to goals the user selected) — v2, once the base data model is proven
- Crafting cost calculator (quantity-weighted TP pricing) — same underlying data, separate feature, v2
- Full achievement/collection catalog — v1 uses a small hand-curated dataset only (see below)
- Zone-based achievement browser
- Squad/commander/role-management tooling
- In-game audio chat, commander tag-up notifications
- Any overlay or game-client-reading functionality (BlishHUD/TaimiHUD territory — different stack, different risk profile, already served)
- Native mobile app (responsive web only)
- Multi-account or guild-level features

## Data Model (Postgres)

- **users** — id, email, password hash (or auth provider id), created_at
- **gw2_api_keys** — id, user_id, encrypted_key, granted_permissions[], created_at
- **items** — id (GW2 item id), name, icon_url, rarity, vendor_value, type, cached_at (lazily populated as items appear in scans, not bulk-fetched)
- **account_snapshots** — user_id, item_id, quantity, location (`bank` / `materials` / `shared_inventory` / `character`), character_name (nullable, set when location = `character`), binding (`none` / `account` / `character` — drives whether "sell on TP" is ever a valid recommendation), scanned_at
- **curated_collections** — id, name, description, item_ids[] (hand-picked for v1 — see "Curated Dataset" below)
- **recipe_cache** — item_id, recipe_ids[] (populated on first lookup via `/v2/recipes/search?input=`, refreshed rarely — this data barely changes)

## Backend Endpoints (Go)

- Auth: signup / login (email+password is fine for v1)
- `POST /api-keys` — accepts a pasted key, validates scopes via `/v2/tokeninfo`, rejects over- or under-permissioned keys with a clear message, stores encrypted
- Scan trigger (internal, session-start-gated) — if snapshot is stale, fetches `/v2/account/inventory` (shared slots), `/v2/account/bank`, `/v2/account/materials`, `/v2/characters` (names only), then `/v2/characters/:id/inventory` for each character; writes all of it to `account_snapshots`
- `GET /inventory` — returns the latest snapshot joined against `items`, `curated_collections`, and `recipe_cache`; this join is the actual product

## Frontend Screens (React/TS)

1. Signup / login
2. Connect API key — explicitly tells the user which permissions to grant and why
3. Results — the main (only) screen: items grouped by location, each row showing name/icon/quantity, tag(s), and recommended action badge; passive "last synced X ago" indicator, no manual scan control

## GW2 API Notes

- Recipes have a **free native reverse lookup**: `/v2/recipes/search?input=<itemID>` returns every recipe using that item as an ingredient. No indexing work needed.
- Achievements/collections have **no reverse lookup**. Collections are achievements with `type: "ItemSet"`. Each achievement has a `bits` array (type: Text/Item/Minipet/Skin) and a `rewards` array — both can reference item IDs, but there's no way to query "which achievements need item X" from the API directly. For v1, this is solved by hand-curating a small dataset rather than indexing the full catalog (see below).
- `/v2/account/inventory` returns **shared inventory slots only** — it is not "the player's inventory." Character bag contents require a separate call per character: `/v2/characters` (list) then `/v2/characters/:id/inventory` for each, which needs the `characters` scope in addition to `account` and `inventories`. `/v2/account/bank` and `/v2/account/materials` are account-wide and don't have this gotcha.
- `/v2/commerce/listings` (not `/v2/commerce/prices`) gives real order-book depth — relevant later for the quantity-weighted cost calculator, not needed for v1.

## Curated Dataset (v1 substitute for full achievement indexing)

Hand-pick ~10-15 collections/legendaries that eat the most inventory space — this is where the pain is sharpest and it's where the "is this for my legendary or not" complaints concentrated. Full catalog indexing (fetch all achievements, parse bits/rewards, build a proper reverse index) is a deferred v2 ETL job, not a v1 blocker. Use dummy/hand-entered data to validate the UX before investing in that pipeline.

## Build Order

1. **Spike (do by hand, not delegated):** confirm the raw API pull against a real personal account — shared inventory, bank, materials, *and* the per-character inventory loop — returns what's expected, including realistic data volume (multiple characters, full bags, several bank tabs). Everything else depends on this assumption holding.
2. Auth + encrypted API key storage + `/v2/tokeninfo` validation
3. Scan pipeline writing `account_snapshots`, with staleness-gated auto-trigger on session start
4. Item metadata caching + recipe reverse-lookup caching
5. Seed curated collections by hand (10-15 entries)
6. Results UI
7. Dogfood on your own account(s), iterate
8. Post back to the original r/Guildwars2 thread for feedback — this closes the loop with the people who described the problem

## Spike Findings (confirmed against a real account)

- All target endpoints work as documented: `/v2/tokeninfo`, `/v2/account/inventory` (shared slots), `/v2/account/bank`, `/v2/account/materials`, `/v2/characters`, `/v2/characters/:id/inventory`, `/v2/recipes/search?input=`.
- Character bags hold the bulk of a real account's items (161 items across characters vs. 2 in shared slots on the test account) — confirms the per-character loop is not optional.
- Items carry a `binding` field (values seen: unbound / Account / Character) and an `upgrades` array (runes/sigils socketed). Binding matters for recommendation logic (see above); upgrade details are out of scope for v1 — store the item id, ignore the rest.
- A single common material can match a large number of recipes (174 for Orichalcum Ore) — display as a count, not a full list, in v1.

## Success Criteria for v1

A real account can be scanned and produce correct, actionable keep/sell/salvage guidance for whatever the curated collections cover. The bar isn't "launch" — it's being able to hand this to the people who complained about bank clutter and have it actually solve their problem.

## Working Notes

- Solo developer, building with Claude Code assistance. No second engineer reviewing diffs — verify what's actually executed rather than trusting a green test run, especially around API key encryption and permission-scope validation, where "it compiled and the happy path worked" can hide a real gap.
- Monetization is intentionally undecided for v1. Treat this as a portfolio piece / community good until there's real usage signal. The metric worth watching isn't revenue, it's return usage — does a user come back to check progress, or connect once and never return.

## Working Preferences
- Claude should not appear as co author in the commits. In general we don't want to show that this is a project written with the help of Claude Code.
- Commits should not be huge so I (the developer) can be able to review them and understand what was done.
- After each commit write in simple words what was done.
- The api key that you can use for testing purposes is: 'BFBA90B9-7E4E-8E47-BE4A-EAD2C813A040D9183995-0BD0-45BD-8035-53E04598FFB5'