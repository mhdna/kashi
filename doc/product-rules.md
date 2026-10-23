# Product rules: discounts, origins, stock kinds

What the owner agreed on 2026-10-23 for filling product fields and for the
Centro discount list. Nothing here is applied on production yet: each change
goes through a dry-run report the owner checks, then a backup, then the run.
Counts are from the Centro till snapshot of 2026-10-17 and are re-counted
against production before anything is written.

How codes are read: the **prefix** is the letters a code starts with
(`CA21LSK2` → `CA`, `EOC5LD7` → `EOC`). "Lady", "Men", "Kids", "Baby" come
from the product name.

## Discount list: Centro

Created on production **inactive**, assigned to the Centro branch (its own
list, so the default list is untouched). Activated only when the owner says.

| # | Group | Asks Summer/Winter? | Winter | Summer |
|---|---|---|---|---|
| 0 | Accessories inside any group below | No | 0% | 0% |
| 1 | Old baby stock: TKS shoes, THP21/THO21 jeans, any TOC | No | 0% | 0% |
| 2 | Baby, T codes (TM6, T61, TJ6) | No | 20% | 20% |
| 3 | Lady Turkey: Lady + T codes (not TOC), all FNV, all FDC (Favori) | No | 30% | 30% |
| 4 | Men jeans, pants only (no jeans jackets, shirts, shorts) | No | 20% | 20% |
| 5 | CA, all | Yes | 20% | 50% |
| 6 | EOC5 lady, EF6 lady | Yes | 20% | 50% |
| 7 | EOC5 / EF6 others (kids, baby, men) | Yes | 0% | 50% |

- Accessories (no discount, never ask a season): hat, cap, glove, scarf,
  shawl, neck cover, sock, collant, bag, wallet, "accessor", bracelet in the
  name. Not "baggy", "set with scarf", "belted", "hair repair".
- Where a product asks Summer/Winter, the season rule wins over groups 2–4.
  An EOC5 men's jeans in winter falls back to the jeans 20%.
- Home products (see below) never get a discount.

Still open (from the owner's own list "Outlet CA/EOC5/EF6 20%, All shoes 20%,
Men summer 50%"):

- Does outlet 20% cover all outlet stock or lady only?
- "All shoes 20%": which products, and does it include old stock?
- Summer 50%: men's summer items only, or all outlet sold as summer?
- C22 is outlet but mostly perfumes: do the perfumes stay out of outlet?
- TB24K2008 and CA23CL2 were caught as accessories by name: confirm.

## Origin

| Origin | Prefixes |
|---|---|
| Turkey | T… clothing codes (TM, T, TNV, TSD, TJ, TDC, TOC, TZ, TMAY, TJU, TMP, TB, TSN, TW, TL, TU, TKS, THV, THP, TP, TV…), FNV, FDC |
| China | COC, CDC, CJ6, X…, GJUL, HBA, CB, OT, digit-only codes, Y, HQ, O, HA, FS, S, I, HD, J, HBB, AO, CF, BL, WX, GH, K, UK, HBC, TEST, TOY, IT, DR, A, ACC, AOC |
| Bangladesh | BOC, N… (N, NSC, NJ, NJUL, NNMAY, NNJUL…), BNV, BM, BC |
| Lebanon | L… (L, LNV, LSEP, LY, LMAY, LJUN), ACNV, ACCJUN, RDC, RJ6, RNV |
| Mixed | OF, OFNV, OFN (Black Friday lots), SU |

## Brand

| Brand | Prefixes |
|---|---|
| Favori | FDC |
| Everywear | R clothing codes: RDC, RJ6, RNV (not RUG, not RS1) |

## Kind of stock

| Kind | Prefixes | What it means |
|---|---|---|
| Outlet | CA (incl. CAMAY, CAJUN, CAJUL, CAAPR), EOC5, EF6, CC, C22 | Summer/Winter is picked at the till when sold; never a fixed season |
| Home | GO, MNU, COM, SPR, BAS, BAR, PILL, HRM, BED, MAT, WP, MAN, KU, GC, MG, WCS, CTS, BS, B, CFS, CS, CP, JP, CH, CAR, RUG, BT | Category Home; never discounted |
| Accessories | ACC, AOC, ACCJUN | Category Accessories; never discounted |
| Shoes | AD | Category Shoes |
| Black Friday lots | OF, OFNV, OFN | Mixed origin |
| Mixed summer | SU | Mixed origin, fixed season Summer |
| Outlet "super cream" | GOJ | Kept sellable until the owner says what "not for sale" means |
| Wholesale by weight | C20, C21, C23, BC2, CB2, L12, L22 "(40 kg)" | Out of scope: retail only |

## Deactivate

Every code starting with **BZ** (96 products: BZ-, BZ100–BZ600, BZD, BZM,
BZX/BZXII), the bazaar lots. The till stops selling them and they leave
search; their sales history stays, and they can be switched back on in kashi.

## Year

From the code for existing products (CA21 → 2021, X24 → 2024, EOC5 → 2025,
EF6 → 2026, TM6 → 2026…). A product created with no year gets the year it was
added. Imported products are not dated by when they were added: the import
put October 2026 on all of them.

## Fields

| Field | Holds | Example |
|---|---|---|
| kind | who it's for | Lady, Men, Kids, Baby |
| category | garment group | Tops, Bottoms, Outerwear, Dresses, Sets, Shoes, Accessories, Underwear & Night, Home |
| sub-category | the garment | Sweater, T-Shirt, Jeans, Coat, Pyjama |
| origin | country | Turkey, China, Bangladesh, Lebanon |
| brand | brand | Favori, Everywear |
| year / season | from the code | 2021…2026 / Summer, Winter |

Only empty fields are filled; nothing the owner set by hand is overwritten.
