#!/usr/bin/env python3
"""Generates the New Product auto-fill rules and code layout.

    python3 db/tools/gen_autofill_rules.py            # prints the rules JSON
    python3 db/tools/gen_autofill_rules.py --sql      # prints the migration body

The rules were worked out from the ~14,000 product codes on production
(2026-11-09); doc/product-rules.md explains the code convention. Rules are
data in app_settings.product_autofill_rules, evaluated by
ui/src/utils/productAutofill.js: per field, the first rule that matches wins.
Edit the tables below, regenerate, and put the output in a new migration.
"""
import json
import sys

# Home goods: never a kind, year or garment from the code.
HOME_PREFIXES = ('GO MNU COM SPR BAS BAR PILL HRM BED MAT WP MAN KU GC MG WCS CTS BS B '
                 'CFS CS CP JP CH CAR RUG BT').split()
HOME = '(?:' + '|'.join(HOME_PREFIXES) + r')(?=\d|-)'

# --- code layout ------------------------------------------------------------
# [origin][month][year][audience][type][price][-suffix]
#   EF6LJE10  = EF, 6 (2026), L lady, JE jeans, price 10
#   CA21LSW   = CA, 21 (2021), L lady, SW sweater (older codes have no price)
#   EOC5BGJ7  = E+OC (October), 5 (2025), BG baby girl, J jacket, price 7
# X, AO, BC and IT codes mix their own model numbers into the same slots, so
# they are left to the name rather than misread.
LAYOUT = (
    r'^(?!(?:X|AO|BC|IT)\d)(?!' + HOME + ')'
    r'(?<origin>[A-Z]+?)'
    r'(?<month>JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|OC|NOV|NV|DEC|DC)?'
    r'(?<year>\d{1,2})'
    r'(?<audience>BB|BG|KB|KG|JB|JG|MX|K|B|L|M)?'
    r'(?<type>[A-Z]+)'
    r'(?<price>\d+(?:[.,]\d+)?)?'
    r'(?:[^A-Z\d].*)?$'
)

rules = []


def rule(source, match, field, value, pattern='', part='', position=1):
    rules.append({
        'source': source, 'match': match, 'position': position,
        'pattern': pattern, 'field': field, 'value': value, 'part': part,
    })


def code_regex(field, value, pattern):
    rule('code', 'regex', field, value, pattern)


def code_part(field, part, pattern, value):
    rule('code', 'part', field, value, pattern, part)


def name_regex(field, value, pattern):
    rule('name', 'regex', field, value, pattern)


def alt(*items):
    return '|'.join(items)


# --- who it's for (kind) ----------------------------------------------------
# A name that says it ("Lady Jeans") beats the one-letter code audience.
KIND_WORDS = [
    ('Baby', r'\b(?:baby|babies|infant|newborn)\b'),
    ('Kids', r'\b(?:kids?|children|junior|boys?|girls?)\b'),
    ('Lady', r'\b(?:women|woman|womens|ladies|lady)\b'),
    ('Men', r'\b(?:men|man|mens)\b'),
    ('Unisex', r'\b(?:mixed|mix|unisex)\b'),
]
for value, pattern in KIND_WORDS:
    name_regex('kind', value, pattern)
for value, pattern in [
    ('Baby', 'BB|BG|B'), ('Kids', 'KB|KG|JB|JG|K'), ('Lady', 'L'),
    ('Men', 'M'), ('Unisex', 'MX'),
]:
    code_part('kind', 'audience', pattern, value)

# --- where it's from (origin) -----------------------------------------------
# By the letters a code starts with (CA21..., EOC5..., X24...). Outlet is the
# retail clearance stock (never a fixed season: the till asks).
ORIGINS = [
    ('Outlet', r'^(?:CA(?:MAY|JUN|JUL|APR)?|EOC|EF|CC|C(?=22))(?=\d)'),
    ('Turkey', r'^(?:T(?!EST|OY)|FNV|FDC|FAVORI)'),
    ('China', r'^(?:COC|CDC|CJ|X(?:APR|MAY|JUN|JUL|AUG)?|GJUL|HBA|HBB|HBC|CB|OT|Y|HQ|O|HA|FS|S|I|HD|J|AO|AOC|CF|BL|WX|GH|K|UK|IT|DR|A|AA|ACC|TOY|\d)(?![A-Z])'),
    ('Bangladesh', r'^(?:N|BOC|BNV|BM|BC)(?![A-Z]{4})'),
    ('Lebanon', r'^(?:L(?:NV|SEP|Y|MAY|JUN|JUL|TB|DC)?|ACNV|ACCJUN|RDC|RJ|RNV)(?=\d)'),
    ('Mixed', r'^(?:OF(?:NV|N)?|SU)(?=\d)'),
]
for value, pattern in ORIGINS:
    code_regex('origin', value, pattern)

# --- brand ------------------------------------------------------------------
code_regex('brand', 'Favori', r'^(?:FDC|FNV)')
code_regex('brand', 'Everywear', r'^(?:RDC|RJ|RNV)(?=\d)')
BRANDS = [
    ('Adidas', 'adidas'), ('Nike', 'nike'), ('PUMA', 'puma'), ('Skechers', 'skechers'),
    ('Lacoste', 'lacoste'), ('Jordan', 'jordan'), ('Louis Vuitton', 'louis vuitton|lv'),
    ('ARMANI', 'armani'), ('BOSS', 'boss'), ('CK', 'ck|calvin klein'), ('D&G', 'd&g|dolce'),
    ('Tommy Hilfiger', 'tommy'), ('Michael Kors', 'michael kors|mk'), ('Chanel', 'chanel'),
    ('Gucci', 'gucci'), ('Crocs', 'crocs'), ('BURBERRY', 'burberry'), ('Timberland', 'timberland'),
    ('New Balance', 'new balance'), ('Clarks', 'clarks'), ('Diesel', 'diesel'), ('Kelme', 'kelme'),
    ('Levis', "levi'?s"), ('Zara', 'zara'), ('Converse', 'converse'), ('Versace', 'versace'),
    ('Prada', 'prada'), ('Disney', 'disney'),
]
for value, words in BRANDS:
    rule('name', 'word', 'brand', value, words)

# --- year -------------------------------------------------------------------
# One digit is the year of the 2020s (EOC5 = 2025, EF6 = 2026); two digits
# are the year itself (CA21 = 2021).
for y in range(2020, 2030):
    d = y % 10
    code_part('year', 'year', f'{d}|{y % 100}' if d >= 5 else str(y % 100), str(y))

# Codes the layout can't read still carry the year: CA21JBSH9L, X24805,
# TMAY25L1411H75 (two digits) and TNV52348, RJ6366 (one digit, 2025/2026).
code_regex('year', '20$1', r'^(?!' + HOME + r')[A-Z]+?(2[0-6])(?=[A-Z\-]|\d)')
code_regex('year', '202$1', r'^(?:T|TNV|TDC|TJ|TL|TI|TK|TG|TM|TSD|TZ|TOC|RDC|RJ|RNV|FDC|CDC|BNV|LSEP|LNV)([5-6])\d')

# --- season -----------------------------------------------------------------
# Outlet stock never gets a fixed season (the till asks), so the month only
# counts for other families; OC is the Oct-batch label (COC5, EOC5), not a
# season. "Winter"/"Summer" in the name always counts.
name_regex('season', 'Winter', r'\bwinter\b')
name_regex('season', 'Summer', r'\bsummer\b')
code_regex('season', 'Summer', r'^(?!(?:CA|EOC|EF|CC|C22))[A-Z]*?(?:MAY|JUN|JUL|AUG)\d')
code_regex('season', 'Winter', r'^(?!(?:CA|EOC|EF|CC|C22))[A-Z]*?(?:NV|NOV|DC|DEC)\d')

# --- garment: category + sub-category ---------------------------------------
# category = garment group, sub-category = the garment. The code's type token
# is read first (very reliable), then words in the name.
code_regex('category', 'Home', r'^(?:' + '|'.join(HOME_PREFIXES) + r')(?=\d|-)')
code_regex('subcategory', 'Home', r'^(?:' + '|'.join(HOME_PREFIXES) + r')(?=\d|-)')
code_regex('category', 'Accessories', r'^(?:ACC|AOC|ACCJUN|ACNV)(?![A-Z])')
code_regex('subcategory', 'Other', r'^(?:ACC|AOC|ACCJUN|ACNV)(?![A-Z])')
code_regex('category', 'Shoes', r'^AD(?=\d)')
code_regex('subcategory', 'Shoes', r'^AD(?=\d)')

# One-digit-year families write S for Set and B for Jacket (EOC5LS..., EF6LB...);
# in two-digit years S is Short.
ONE_DIGIT = r'^[A-Z]+?\d(?!\d)(?:BB|BG|KB|KG|JB|JG|MX|K|B|L|M)?'
for field, value in (('category', 'Sets'), ('subcategory', 'Set')):
    code_regex(field, value, ONE_DIGIT + r'S(?:\d|[^A-Z\d]|$)')
for field, value in (('category', 'Outerwear'), ('subcategory', 'Jacket')):
    code_regex(field, value, ONE_DIGIT + r'B(?:\d|[^A-Z\d]|$)')

# (category, sub-category, type tokens)
TYPES = [
    ('Tops', 'Sweatshirt', 'SWE SPSWE FSWE CGSWE SWT SSWT FSWT HZSWT SWECP RSWE SPSW JH'),
    ('Tops', 'Sweater', 'SW CGSW CG BO FSW WSW NSW SWJB SWJG SWBB SWKG CA'),
    ('Tops', 'Hoodie', 'H HO HD HOCP FHO SPH'),
    ('Tops', 'Shirt', 'SH SHL SHS WSH WSHL SHX SHXS SHXL JESH ESH SHFBC SHHBC SHNBC'),
    ('Tops', 'T-Shirt', 'TSH SPTSH SPISH ISH STSH BDY TSHX TSHXXL TSHB TSHBC BTSH CTSH IS ISHX NFTS TS'),
    ('Tops', 'Polo', 'PO PL POX POL PTSH'),
    ('Tops', 'Blouse', 'BL BLS BLL LBL IBL XLBL LBSS BSS BLO LBLS'),
    ('Tops', 'Body', 'BLH'),
    ('Outerwear', 'Jacket', 'JK J SJ SPJK FJK FJ ANO AN JEJK JKC JKSP JJB JKD JKG JKL JKB LJK RJK JEJ ZJ ANJ JJK JJG AJ DFJ SKIJ SKP SKIP SKPN JSKPN TBZ'),
    ('Outerwear', 'Coat', 'COAT CO VZ WVZ FUR WV CT LCO FU'),
    ('Outerwear', 'Vest & Gilet', 'V VS SPV SPVS FV VKG'),
    ('Bottoms', 'Pants', 'PN P CTPN WCTPN SPPN CTP JGPN JOP JO CP WPN FPN HA HAPN WP CRP CRPN SP PA PNT CPN PNXL PNX RPN THPN MXPN KP FP FJO WLP CLP LIP VP SYP'),
    ('Bottoms', 'Jeans', 'JE JN FJE WJE ZJE JEXL JEX SLJE SKN GH TOROS DOGA GR IRA'),
    ('Bottoms', 'Shorts', 'S CTS SPS WS SHO CSHO'),
    ('Bottoms', 'Skirt', 'SK WSK'),
    ('Bottoms', 'Leggings', 'LEG LE LEP'),
    ('Dresses', 'Dress', 'DR D WDR LD PDR MD ABY WD ABAYA DRBG DRKG PADR'),
    ('Sets', 'Set', 'SET WSET SPSET FSET KSET GSET FS HS BS LIS'),
    ('Sets', 'Overall & Romper', 'OV WOV SAL SKOV FOV SPOV OVBG OVBB JSKOV SKIOV RB'),
    ('Sleepwear', 'Pyjama', 'PJS HPJS CPJS LPJS PJSET NTP NSET NDR NPN NS NSO NLN NTSH NTPN NROB'),
    ('Sleepwear', 'Robe', 'ROB FROB RO'),
    ('Underwear', 'Underwear', 'BX UNBR SLIP UNSL SPBR UN UNST BR BRA UNBX UNGA UNB BRE GA RA RT UNBRL UNTSH'),
    ('Swimwear', 'Swimwear', 'MAYO SWMY MY SWBR SWSL BU GBU MAYOTP MAYOSET MAYOSL'),
    ('Accessories', 'Socks & Tights', 'SOCKS SO CL COL ASO STSO'),
    ('Accessories', 'Hat & Cap', 'HAT WHT WCAP CAP HT KD HATSET HSET WHTSET WHA WH'),
    ('Accessories', 'Scarf & Shawl', 'SHW SCARF SCA SHA NC PSHA'),
    ('Accessories', 'Gloves', 'GL DZ'),
    ('Accessories', 'Bag', 'ITBA CM BAG AG IBA ICBA IPTBA BKPK KPK BA BALV'),
    ('Accessories', 'Belt', 'BE BELT'),
    ('Shoes', 'Shoes', 'SS SHE WSS SNE SKE FTW WSLP RSS SSHH LBO RVBO VBO BAL BBUG FBUG LBOT RLBO BLE'),
    ('Toys', 'Toys', 'THD TTHD'),
    ('Beauty', 'Perfume & Care', 'PF CS'),
    ('Home', 'Home', 'HHR TWL TW TWS GPC CUPSET GPSET'),
]
for category, sub, tokens in TYPES:
    pattern = alt(*tokens.split())
    code_part('category', 'type', pattern, category)
    code_part('subcategory', 'type', pattern, sub)

# Words in the name, first match wins (top to bottom), for codes the layout
# can't read. Same order and words as doc/product-rules.md.
NAME_TYPES = [
    ('Home', 'Home', r'\b(?:cups?|glass(?:es)?|plates?|bowls?|towels?|blankets?|table ?cover|carpets?|cutting board|mats?|pots?|trays?|containers?|mugs?|sheets?|pillows?|comforters?|curtains?|duvets?|bed set|tables?|cabinets?)\b'),
    ('Beauty', 'Perfume & Care', r'\b(?:body (?:milk|lotion)|perfumes?|for (?:him|her)|shampoo|conditioner|lotion|cream|sanitizer|shower gel|spray)\b'),
    ('Accessories', 'Bag', r'\b(?:bags?|sacks?|wallets?|purse|backpacks?)\b'),
    ('Accessories', 'Hat & Cap', r'\b(?:hats?|caps?|beanie)\b'),
    ('Accessories', 'Gloves', r'\bgloves?\b'),
    ('Accessories', 'Scarf & Shawl', r'\b(?:scarf|scarves|shawls?|neck cover|hijab)\b'),
    ('Accessories', 'Socks & Tights', r'\b(?:socks?|collants?|tights)\b'),
    ('Accessories', 'Belt', r'\bbelts?\b'),
    ('Accessories', 'Watch & Jewelry', r'\b(?:watch|bracelet|rings?|necklace|earrings?|key holder)\b'),
    ('Accessories', 'Other', r'\b(?:accessor\w*|sunglasses|umbrella|lunch box|water bottle)\b'),
    ('Shoes', 'Shoes', r'\b(?:shoes?|sneakers?|boots?|slippers?|sandals?|trainers?)\b'),
    ('Swimwear', 'Swimwear', r'\b(?:mayo|swim\w*|bikini|burkini)\b'),
    ('Sleepwear', 'Pyjama', r'\b(?:pyjamas?|pajamas?|night)\b'),
    ('Sleepwear', 'Robe', r'\b(?:robe|bath ?robe)\b'),
    ('Underwear', 'Underwear', r'\b(?:boxers?|slips?|bra|underwear|bretel|briefs?|camisole)\b'),
    ('Sets', 'Overall & Romper', r'\b(?:overalls?|romper|barboteuse|jumpsuit|onesie|salopette)\b'),
    ('Sets', 'Set', r'(?:\bset\b|\bsuit\b|tracksuit|\b\d+ ?pcs\b)'),
    ('Dresses', 'Dress', r'\b(?:dress(?:es)?|abaya|tunic|kaftan)\b'),
    ('Outerwear', 'Coat', r'\b(?:coats?|parka|trench|fur|vizon)\b'),
    ('Outerwear', 'Jacket', r'\b(?:jackets?|anorak|blazer|bomber|windbreaker|ski)\b'),
    ('Outerwear', 'Vest & Gilet', r'\b(?:vests?|gilet)\b'),
    ('Tops', 'Hoodie', r'\b(?:hood\w*|hoddie)\b'),
    ('Tops', 'Sweatshirt', r'\bsweatshirts?\b'),
    ('Tops', 'Sweater', r'\b(?:sweaters?|pullover|cardigan|knit|wool|jumper|bolero)\b'),
    ('Tops', 'Polo', r'\bpolo\b'),
    ('Tops', 'T-Shirt', r'\b(?:t-?shirts?|tee|tops?|tank|sleeveless|carre)\b'),
    ('Tops', 'Blouse', r'\b(?:blouse|blouze)\b'),
    ('Tops', 'Shirt', r'\bshirts?\b'),
    ('Tops', 'Body', r'\bbody\b'),
    ('Bottoms', 'Jeans', r'\b(?:jeans?|denim)\b'),
    ('Bottoms', 'Shorts', r'\b(?:shorts?|bermuda)\b'),
    ('Bottoms', 'Skirt', r'\bskirts?\b'),
    ('Bottoms', 'Leggings', r'\bleggings?\b'),
    ('Bottoms', 'Pants', r'\b(?:pants?|trousers?|jogging|cargo|chino)\b'),
    ('Toys', 'Toys', r'\b(?:toys?|football|basketball|games?|balls?|dolls?|puzzle|lego|bike|scooter)\b'),
]
for category, sub, pattern in NAME_TYPES:
    name_regex('category', category, pattern)
    name_regex('subcategory', sub, pattern)

# --- price ------------------------------------------------------------------
# Only the price part of a code the layout reads (EF6LJE10 -> 10). A bare
# trailing number is a model number (T1178, X2527385), not a price.
rule('code', 'part', 'price', '', '', 'price')

if __name__ == '__main__':
    if '--sql' in sys.argv:
        body = json.dumps(rules, indent=2, ensure_ascii=False)
        print(f"UPDATE app_settings SET\n  product_autofill_code_layout = $layout${LAYOUT}$layout$,\n"
              f"  product_autofill_rules = $rules${body}$rules$\nWHERE id = 1;")
    else:
        print(json.dumps({'layout': LAYOUT, 'rules': rules}, ensure_ascii=False))
