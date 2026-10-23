# Product rules: discounts, origins, stock kinds

What the owner agreed on 2026-10-23 for filling product fields and for the
Centro discount list.

Applied on production 2026-10-23 (backup first:
`/root/backups/kashi-before-attr-backfill-2026-10-23.dump`): origin, brand,
year, kind, category, sub-category and season filled where empty (54,157
fields on 13,650 products), and the 114 BZ products and their variants made
inactive. Skipped: names with "kg" (wholesale, 211) and TEST codes. The
Centro discount list is not created yet.
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
| From the name | Adidas, Nike, PUMA, Skechers, Lacoste, Jordan, Louis Vuitton (also "LV"), ARMANI, BOSS, CK, D&G (also "Dolce & Gabbana"), Tommy Hilfiger ("Tommy"), Michael Kors ("MK"), Chanel, Gucci, Crocs… — filled 2026-10-23 on 118 products. Existing spellings (ARMANI, BOSS, PUMA) kept. Cartoon characters (Pokemon, Frozen, Spiderman) are not brands |

A country is never a brand: three LSEP shoes had "Lebanon" typed as their
brand; it was cleared (they already had origin Lebanon).

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
| Wholesale by weight | any name with "kg" (mostly L12, C22, L22, BC21) | Out of scope: retail only, never filled |

## Deactivate

Every code starting with **BZ** (96 products: BZ-, BZ100–BZ600, BZD, BZM,
BZX/BZXII), the bazaar lots. The till stops selling them and they leave
search; their sales history stays, and they can be switched back on in kashi.

## Year

A year is read only where the digits after the prefix are one: two digits
20–26 (CA21, X24, N22, GO22) or one digit 5/6 for the families that write the
year as one digit (COC5, EOC5, EF6, BOC5, TOC5, TNV5…, TJ6, TM6, LSEP6).
Model numbers are left alone (TW01, TP10495, TKS001, AD001, LTB29, home
codes like P2000).

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

Kind comes from the name's first word: Lady, Men/Man, Kids, Baby; Junior →
Kids, Mixed/Mix → Unisex.

## Categories

Read from words in the product name, first match wins, top to bottom. Home
prefixes (see Kind of stock) are always Home whatever the name says. BZ is
left out (deactivated). Coverage on the 2026-10-17 snapshot: 13,173 of
13,755 products (96%).

| Category | Sub-category | Name words |
|---|---|---|
| Home | Home | cup, glass, plate, bowl, towel, blanket, table cover, carpet, cutting board, mat, pot, tray, container, mug, sheet, pillow, comforter, curtain |
| Beauty | Perfume & Care | body milk, body lotion, perfume, for him/her, shampoo, conditioner, lotion, cream, sanitizer, shower gel, spray |
| Accessories | Bag | bag, sack, wallet, purse, backpack |
| | Hat & Cap | hat, cap, beanie |
| | Gloves | glove |
| | Scarf & Shawl | scarf, shawl, neck cover, hijab |
| | Socks & Tights | sock, collant, tights |
| | Belt | belt |
| | Watch & Jewelry | watch, bracelet, ring, necklace, earring, key holder |
| | Other | accessor, sunglasses, umbrella, lunch box, water bottle |
| Shoes | Shoes | shoe, sneaker, boot, slipper, sandal, trainer |
| Swimwear | Swimwear | mayo, swim, bikini, burkini |
| Underwear & Night | Pyjama | pyjama, night |
| | Underwear | boxer, slip, bra, underwear, bretel, brief, camisole |
| | Robe | rob, robe |
| Sets | Overall & Romper | overall, romper, barboteuse, jumpsuit, onesie, salopette |
| | Set | set, suit, tracksuit, "N pcs" |
| Dresses | Dress | dress, abaya, tunic, kaftan |
| Outerwear | Coat | coat, parka, trench, fur, vizon |
| | Jacket | jacket, anorak, blazer, bomber, windbreaker, ski |
| | Vest & Gilet | vest, gilet |
| Tops | Hoodie | hood |
| | Sweatshirt | sweatshirt |
| | Sweater | sweater, pullover, cardigan, knit, wool, jumper, bolero |
| | Polo | polo |
| | T-Shirt | t-shirt, tee, top, tank, sleeveless, carre |
| | Blouse | blouse, blouze |
| | Shirt | shirt |
| | Body | body |
| Bottoms | Jeans | jean, denim |
| | Shorts | short, bermuda |
| | Skirt | skirt |
| | Leggings | legging |
| | Pants | pant, trouser, jogging, cargo, chino |
| Toys | Toys | toy, football, basketball, rc, game, ball, doll, puzzle, lego, bike, scooter… |

Left without a category (582): names that say nothing about the garment —
"Baby Item" (131), "Unidentified", "tmp", "Mixed Items", "Sports articles".
Their kind is still filled from the name.
