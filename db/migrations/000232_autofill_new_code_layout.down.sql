-- Back to the rules and layout production had before 000232 (the 000122
-- starting rules plus the edits made on the Auto-fill Rules page).
UPDATE app_settings SET
  product_autofill_code_layout = $layout$^(?<origin>[A-Z]+?)(?<month>JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|OC|NOV|NV|DEC|DC)?(?<year>\d{1,2})(?<audience>BB|BG|KB|KG|JB|JG|MX|K|B|L|M)?(?<type>[A-Z]+)(?<price>\d+(?:[.,]\d+)?)(?:\D.*)?$$layout$,
  product_autofill_rules = $rules$[
 {
  "part": "",
  "field": "origin",
  "match": "char",
  "value": "Lebanon",
  "source": "code",
  "pattern": "L",
  "position": 1
 },
 {
  "part": "",
  "field": "origin",
  "match": "char",
  "value": "Turkey",
  "source": "code",
  "pattern": "T",
  "position": 1
 },
 {
  "part": "year",
  "field": "year",
  "match": "part",
  "value": "2020",
  "source": "code",
  "pattern": "20",
  "position": 1
 },
 {
  "part": "year",
  "field": "year",
  "match": "part",
  "value": "2021",
  "source": "code",
  "pattern": "21",
  "position": 1
 },
 {
  "part": "year",
  "field": "year",
  "match": "part",
  "value": "2022",
  "source": "code",
  "pattern": "22",
  "position": 1
 },
 {
  "part": "year",
  "field": "year",
  "match": "part",
  "value": "2023",
  "source": "code",
  "pattern": "23",
  "position": 1
 },
 {
  "part": "year",
  "field": "year",
  "match": "part",
  "value": "2024",
  "source": "code",
  "pattern": "24",
  "position": 1
 },
 {
  "part": "year",
  "field": "year",
  "match": "part",
  "value": "2025",
  "source": "code",
  "pattern": "5|25",
  "position": 1
 },
 {
  "part": "year",
  "field": "year",
  "match": "part",
  "value": "2026",
  "source": "code",
  "pattern": "6|26",
  "position": 1
 },
 {
  "part": "year",
  "field": "year",
  "match": "part",
  "value": "2027",
  "source": "code",
  "pattern": "7|27",
  "position": 1
 },
 {
  "part": "year",
  "field": "year",
  "match": "part",
  "value": "2028",
  "source": "code",
  "pattern": "8|28",
  "position": 1
 },
 {
  "part": "year",
  "field": "year",
  "match": "part",
  "value": "2029",
  "source": "code",
  "pattern": "9|29",
  "position": 1
 },
 {
  "part": "month",
  "field": "season",
  "match": "part",
  "value": "Winter",
  "source": "code",
  "pattern": "OC|OCT|NV|NOV|DC|DEC",
  "position": 1
 },
 {
  "part": "month",
  "field": "season",
  "match": "part",
  "value": "Summer",
  "source": "code",
  "pattern": "MAY|JUN|JUL|AUG",
  "position": 1
 },
 {
  "part": "audience",
  "field": "category",
  "match": "part",
  "value": "Baby",
  "source": "code",
  "pattern": "BB",
  "position": 1
 },
 {
  "part": "audience",
  "field": "subcategory",
  "match": "part",
  "value": "Boys",
  "source": "code",
  "pattern": "BB",
  "position": 1
 },
 {
  "part": "audience",
  "field": "category",
  "match": "part",
  "value": "Baby",
  "source": "code",
  "pattern": "BG",
  "position": 1
 },
 {
  "part": "audience",
  "field": "subcategory",
  "match": "part",
  "value": "Girls",
  "source": "code",
  "pattern": "BG",
  "position": 1
 },
 {
  "part": "audience",
  "field": "category",
  "match": "part",
  "value": "Kids",
  "source": "code",
  "pattern": "KB",
  "position": 1
 },
 {
  "part": "audience",
  "field": "subcategory",
  "match": "part",
  "value": "Boys",
  "source": "code",
  "pattern": "KB",
  "position": 1
 },
 {
  "part": "audience",
  "field": "category",
  "match": "part",
  "value": "Kids",
  "source": "code",
  "pattern": "KG",
  "position": 1
 },
 {
  "part": "audience",
  "field": "subcategory",
  "match": "part",
  "value": "Girls",
  "source": "code",
  "pattern": "KG",
  "position": 1
 },
 {
  "part": "audience",
  "field": "category",
  "match": "part",
  "value": "Kids",
  "source": "code",
  "pattern": "JB",
  "position": 1
 },
 {
  "part": "audience",
  "field": "subcategory",
  "match": "part",
  "value": "Boys",
  "source": "code",
  "pattern": "JB",
  "position": 1
 },
 {
  "part": "audience",
  "field": "category",
  "match": "part",
  "value": "Kids",
  "source": "code",
  "pattern": "JG",
  "position": 1
 },
 {
  "part": "audience",
  "field": "subcategory",
  "match": "part",
  "value": "Girls",
  "source": "code",
  "pattern": "JG",
  "position": 1
 },
 {
  "part": "audience",
  "field": "category",
  "match": "part",
  "value": "Kids",
  "source": "code",
  "pattern": "K",
  "position": 1
 },
 {
  "part": "audience",
  "field": "category",
  "match": "part",
  "value": "Baby",
  "source": "code",
  "pattern": "B",
  "position": 1
 },
 {
  "part": "audience",
  "field": "category",
  "match": "part",
  "value": "Lady",
  "source": "code",
  "pattern": "L",
  "position": 1
 },
 {
  "part": "audience",
  "field": "category",
  "match": "part",
  "value": "Men",
  "source": "code",
  "pattern": "M",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "T-Shirt",
  "source": "code",
  "pattern": "TSH",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Sport T-Shirt",
  "source": "code",
  "pattern": "SPTSH|STSH",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "I-Shirt",
  "source": "code",
  "pattern": "ISH",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Shirt",
  "source": "code",
  "pattern": "SH",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Shirt L/S",
  "source": "code",
  "pattern": "SHL",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Shirt S/S",
  "source": "code",
  "pattern": "SHS",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Winter Shirt",
  "source": "code",
  "pattern": "WSH",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Winter Shirt L/S",
  "source": "code",
  "pattern": "WSHL",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Polo",
  "source": "code",
  "pattern": "PO",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Jeans",
  "source": "code",
  "pattern": "JE",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Sweater",
  "source": "code",
  "pattern": "SW",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Cardigan Sweater",
  "source": "code",
  "pattern": "CGSW",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Sweatshirt",
  "source": "code",
  "pattern": "SWE|SWT",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Hoodie",
  "source": "code",
  "pattern": "H|HO",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Jacket",
  "source": "code",
  "pattern": "J|JK",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Sport Jacket",
  "source": "code",
  "pattern": "SJ|SPJK",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Fleece Jacket",
  "source": "code",
  "pattern": "FJ|FJK",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Anorak",
  "source": "code",
  "pattern": "AN|ANO",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Coat",
  "source": "code",
  "pattern": "COAT|CO",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Vizon",
  "source": "code",
  "pattern": "VZ",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Winter Vizon",
  "source": "code",
  "pattern": "WVZ",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Vest",
  "source": "code",
  "pattern": "V|VS",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Blazer",
  "source": "code",
  "pattern": "B|BZ",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Bolero",
  "source": "code",
  "pattern": "BO",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Cardigan",
  "source": "code",
  "pattern": "CA",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Blouse",
  "source": "code",
  "pattern": "BL|BLS",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Blouse L/S",
  "source": "code",
  "pattern": "BLL",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Long Blouse",
  "source": "code",
  "pattern": "LBL",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Pants",
  "source": "code",
  "pattern": "P|PN|PA",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Cotton Pants",
  "source": "code",
  "pattern": "CP|CTPN",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Winter Cotton Pants",
  "source": "code",
  "pattern": "WCTPN",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Winter Pants",
  "source": "code",
  "pattern": "WP",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Jogging Pants",
  "source": "code",
  "pattern": "JOP|JGPN",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Sport Pants",
  "source": "code",
  "pattern": "SPPN|SP",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Cargo Pants",
  "source": "code",
  "pattern": "CRP",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Fleece Pants",
  "source": "code",
  "pattern": "FPN",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Hawaii Pants",
  "source": "code",
  "pattern": "HA",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Legging",
  "source": "code",
  "pattern": "LEG|LE",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Collant",
  "source": "code",
  "pattern": "COL",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "regex",
  "value": "Set",
  "source": "code",
  "pattern": "^[A-Z]+\\d(?:BB|BG|KB|KG|JB|JG|K|B|L|M)?S\\d",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Short",
  "source": "code",
  "pattern": "S|SHO",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Cotton Short",
  "source": "code",
  "pattern": "CTS",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Sport Short",
  "source": "code",
  "pattern": "SPS",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Skirt",
  "source": "code",
  "pattern": "SK",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Winter Skirt",
  "source": "code",
  "pattern": "WSK",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Dress",
  "source": "code",
  "pattern": "DR|D",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Winter Dress",
  "source": "code",
  "pattern": "WDR",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Overall",
  "source": "code",
  "pattern": "OV",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Winter Overall",
  "source": "code",
  "pattern": "WOV",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Ski Overall",
  "source": "code",
  "pattern": "SKOV",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Pyjama Set",
  "source": "code",
  "pattern": "PJS",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Socks",
  "source": "code",
  "pattern": "SO",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Gloves",
  "source": "code",
  "pattern": "GL",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Hat",
  "source": "code",
  "pattern": "HAT",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Wool Hat",
  "source": "code",
  "pattern": "WHT",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Scarf",
  "source": "code",
  "pattern": "SCA",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Shawl",
  "source": "code",
  "pattern": "SHA|SHW",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Slip",
  "source": "code",
  "pattern": "SL",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Mayo Slip",
  "source": "code",
  "pattern": "MAYOSL",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Mayo Bra",
  "source": "code",
  "pattern": "MAYOBR",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Bretel",
  "source": "code",
  "pattern": "BR",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Shoes",
  "source": "code",
  "pattern": "SHE",
  "position": 1
 },
 {
  "part": "type",
  "field": "type",
  "match": "part",
  "value": "Sneakers",
  "source": "code",
  "pattern": "SNE",
  "position": 1
 },
 {
  "part": "price",
  "field": "price",
  "match": "part",
  "value": "",
  "source": "code",
  "pattern": "",
  "position": 1
 },
 {
  "part": "",
  "field": "price",
  "match": "trailing_number",
  "value": "",
  "source": "code",
  "pattern": "",
  "position": 1
 },
 {
  "part": "",
  "field": "category",
  "match": "word",
  "value": "Kids",
  "source": "name",
  "pattern": "kids|kid|children",
  "position": 1
 },
 {
  "part": "",
  "field": "category",
  "match": "word",
  "value": "Baby",
  "source": "name",
  "pattern": "baby|babies|infant|newborn",
  "position": 1
 },
 {
  "part": "",
  "field": "category",
  "match": "word",
  "value": "Lady",
  "source": "name",
  "pattern": "women|woman|ladies|lady",
  "position": 1
 },
 {
  "part": "",
  "field": "category",
  "match": "word",
  "value": "Men",
  "source": "name",
  "pattern": "men|man",
  "position": 1
 },
 {
  "part": "",
  "field": "subcategory",
  "match": "word",
  "value": "Boys",
  "source": "name",
  "pattern": "boy|boys",
  "position": 1
 },
 {
  "part": "",
  "field": "subcategory",
  "match": "word",
  "value": "Girls",
  "source": "name",
  "pattern": "girl|girls",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Sweatshirt",
  "source": "name",
  "pattern": "sweatshirt|sweatshirts",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Hoodie",
  "source": "name",
  "pattern": "hoodie|hoodies|hooded",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "T-Shirt",
  "source": "name",
  "pattern": "t-shirt|tshirt|tee",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Shirt",
  "source": "name",
  "pattern": "shirt|shirts",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Sweater",
  "source": "name",
  "pattern": "sweater|pullover",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Jacket",
  "source": "name",
  "pattern": "jacket|jackets",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Coat",
  "source": "name",
  "pattern": "coat|coats",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Jeans",
  "source": "name",
  "pattern": "jeans|jean",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Pants",
  "source": "name",
  "pattern": "pants|pant|trousers",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Short",
  "source": "name",
  "pattern": "shorts|short",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Dress",
  "source": "name",
  "pattern": "dress|dresses",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Skirt",
  "source": "name",
  "pattern": "skirt|skirts",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Overall",
  "source": "name",
  "pattern": "overall|overalls",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Pyjama Set",
  "source": "name",
  "pattern": "pyjama|pyjamas|pajama|pajamas",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Socks",
  "source": "name",
  "pattern": "socks|sock",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Polo",
  "source": "name",
  "pattern": "polo",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Vest",
  "source": "name",
  "pattern": "vest",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Blouse",
  "source": "name",
  "pattern": "blouse|blouze",
  "position": 1
 },
 {
  "part": "",
  "field": "type",
  "match": "word",
  "value": "Legging",
  "source": "name",
  "pattern": "legging|leggings",
  "position": 1
 }
]$rules$
WHERE id = 1;
