import html, re, sys, os
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from content import *
from content_more import WIZARD_EXTRA
from content_odoo import *
import diagrams as D
import glob
SHOT_DIR_REL = "../../_local/wizard-screenshots-2026-10-31"
SHOT_DIR = os.path.normpath(os.path.join(sys.argv[1], SHOT_DIR_REL))
SHOTS = {  # milestone -> filename patterns (substring match) of Wizard screenshots that show how they do it
    "M0": ["sales-sales-index__record.png", "sales-sales-index__record-actions-menu", "sales-offer.png", "sales-offer__actions-menu", "sales-order-index__actions-menu"],
    "M1": ["sales-sales-indexaging", "sales-sales-indexnotdue", "receivable-receipt-index", "related-Statement_of_Account", "related-Client_Balance", "related-Unpaid_Invoices", "auxiliary-client.png", "auxiliary-client-list"],
    "M2": ["sales-purchase-index__record.png", "sales-purchase-index__record-actions-menu", "sales-porder-index", "sales-purchase-indexaging", "sales-expensesforgoods-index", "procurment-request-index", "receivable-payment-index", "receivable-paymentreq-index"],
    "M3": ["inventory-count-list", "inventory-count-listdetail", "inventory-wzinv-main", "inventory-wzinv-costsindex", "inventory-internalrequisition", "product-product.png", "product-product__tab", "product-product-listpricelist"],
    "M4": ["sales-offer", "sales-order-index", "sales-order-listorder", "sales-order-listproductorder", "shipment-shipment"],
    "M5": ["account-account.png", "account-voucher-index__record.png", "account-voucher-index__actions-menu", "account-statement", "global-costCenter", "global-initialize-index1", "global-tools-indexclosinginventory"],
    "M6": ["account-bank-list", "account-checkbook", "account-CheckMngRec", "account-bankOp", "account-AccountReconciliation", "global-vat", "global-CurrencyRate", "global-currency"],
    "M7": ["global-approval", "global-audit", "auxiliary-users-list", "global-wzOptions"],
    "M8": ["payroll-attendance-indexleave", "payroll-payroll"],
    "M13": ["sales-assets-index", "maintenance-maintenance-index", "maintenance-tickets", "production-production"],
}
def shots_for(mid):
    files = sorted(glob.glob(os.path.join(SHOT_DIR, "*.png")))
    picked = []
    for pat in SHOTS.get(mid, []):
        for f in files:
            b = os.path.basename(f)
            if pat in b and b not in picked:
                picked.append(b)
    return picked[:10]
MATRIX = MATRIX + WIZARD_EXTRA
TITLE = 'kashi vs Wizard Cloud ERP & Odoo: missing features & milestone roadmap'

OUT = sys.argv[1]
STATUS_LABEL = {"ahead": "kashi ahead", "have": "have", "partial": "partial", "missing": "missing", "skip": "skip"}
STATUS_ICON = {"ahead": "🟢", "have": "✅", "partial": "🟡", "missing": "❌", "skip": "⚪"}
PRIO_TEXT = {"P0": "P0 — needed to drop Wizard in our own business",
             "P1": "P1 — needed to sell kashi to other retailers/wholesalers",
             "P2": "P2 — needed to compete across the board",
             "P3": "P3 — only for specific verticals / on demand"}
SIZE_TEXT = {"S": "< 1 week", "M": "1–3 weeks", "L": "3–6 weeks", "XL": "6+ weeks"}

counts = {k: 0 for k in STATUS_LABEL}
for _, rows in MATRIX:
    for _, st, _ in rows:
        counts[st] += 1
total = sum(counts.values())
ocounts = {k: 0 for k in STATUS_LABEL}
for _, rows in ODOO_MATRIX:
    for _, st, _ in rows:
        ocounts[st] += 1
ototal = sum(ocounts.values())
nfeat = sum(len(m["features"]) for m in MILESTONES)

# ------------------------------------------------------------------ Markdown
md = []
w = md.append
w(f"# {TITLE}\n")
w(f"_Last updated {DATE}._\n")
w("## How this was researched\n")
for s in SOURCES: w(f"- {s}")
w("\n## TL;DR\n")
for s in TLDR: w(f"- {s}")
w("\n## Scoreboard\n")
w(f"Of **{total}** capabilities Wizard has (from their app and their website):\n")
w("| Status | Count | Meaning |\n|---|---|---|")
meaning = {"ahead": "kashi already does it better", "have": "kashi has it", "partial": "kashi has a piece of it",
           "missing": "kashi doesn't have it", "skip": "not worth building for us"}
for k in STATUS_LABEL:
    w(f"| {STATUS_ICON[k]} {STATUS_LABEL[k]} | {counts[k]} | {meaning[k]} |")
w("\n## Priorities & sizes used below\n")
for k, v in PRIO_TEXT.items(): w(f"- **{v}**")
w("- Sizes are rough focused-work estimates: " + ", ".join(f"**{k}** {v}" for k, v in SIZE_TEXT.items()) + ".\n")
w("## Milestone overview\n")
w("| # | Milestone | Priority | Size | Goal |\n|---|---|---|---|---|")
for m in MILESTONES:
    w(f"| {m['id']} | [{m['title']}](#{m['id'].lower()}) | {m['priority']} | {m['size']} | {m['goal']} |")
w("\n### Suggested order\n")
for s in ORDER_NOTES: w(f"- {s}")
w("\n## Where kashi already wins (amplify these)\n")
for t, d in EDGES: w(f"- **{t}**: {d}")
w("\n## What Wizard is set up with for R1 today\n")
w("| Area | What we found |\n|---|---|")
for a, b in R1_SETUP: w(f"| {a} | {b} |")
w("\n## Full feature matrix: Wizard vs kashi\n")
w("Legend: " + " · ".join(f"{STATUS_ICON[k]} {STATUS_LABEL[k]}" for k in STATUS_LABEL) + "\n")
for mod, rows in MATRIX:
    c = {k: sum(1 for r in rows if r[1] == k) for k in STATUS_LABEL}
    summary = ", ".join(f"{c[k]} {STATUS_LABEL[k]}" for k in STATUS_LABEL if c[k])
    w(f"### {mod}\n\n_{summary}_\n")
    w("| Wizard capability | kashi | Notes |\n|---|---|---|")
    for f, st, note in rows:
        w(f"| {f} | {STATUS_ICON[st]} {STATUS_LABEL[st]} | {note} |")
    w("")
w("## Milestones in detail\n")
for m in MILESTONES:
    w(f"<a id=\"{m['id'].lower()}\"></a>\n### {m['id']} · {m['title']}\n")
    w(f"**Priority:** {m['priority']} · **Size:** {m['size']} ({SIZE_TEXT[m['size']]})\n")
    w(f"**Goal:** {m['goal']}\n")
    w(f"**Why:** {m['why']}\n")
    w("| Feature | Priority | Size | What to build |\n|---|---|---|---|")
    for name, p, s, d in m["features"]:
        w(f"| **{name}** | {p} | {s} | {d} |")
    if m["done_when"]:
        w(f"\n**Done when:** {m['done_when']}\n")
    w("")
w("## Part 2 — kashi vs Odoo\n")
for s_ in ODOO_INTRO: w(f"- {s_}")
w(f"\nOf **{ototal}** Odoo capabilities: " + ", ".join(f"{STATUS_ICON[k]} {ocounts[k]} {STATUS_LABEL[k]}" for k in STATUS_LABEL) + ".\n")
for mod, rows in ODOO_MATRIX:
    w(f"### Odoo · {mod}\n")
    w("| Odoo capability | kashi | Notes |\n|---|---|---|")
    for f, st, note in rows:
        w(f"| {f} | {STATUS_ICON[st]} {STATUS_LABEL[st]} | {note} |")
    w("")
w("### Odoo-driven milestones (O-series)\n")
w("These come on top of M0–M13; where an M-milestone already covers an Odoo feature it is not repeated.\n")
w("| # | Milestone | Priority | Size | Goal |\n|---|---|---|---|---|")
for m in ODOO_MILESTONES:
    w(f"| {m['id']} | [{m['title']}](#{m['id'].lower()}) | {m['priority']} | {m['size']} | {m['goal']} |")
w("")
for m in ODOO_MILESTONES:
    w(f"<a id=\"{m['id'].lower()}\"></a>\n#### {m['id']} · {m['title']}\n")
    w(f"**Priority:** {m['priority']} · **Size:** {m['size']} ({SIZE_TEXT[m['size']]})\n")
    w(f"**Goal:** {m['goal']}\n\n**Why:** {m['why']}\n")
    w("| Feature | Priority | Size | What to build |\n|---|---|---|---|")
    for name, p_, sz, d in m["features"]:
        w(f"| **{name}** | {p_} | {sz} | {d} |")
    if m["done_when"]: w(f"\n**Done when:** {m['done_when']}\n")
    w("")
w("## Part 3 — What matters most in the Lebanese market\n")
w("| Need | Importance | Wizard | Odoo | kashi today | What to build |\n|---|---|---|---|---|---|")
for need, imp, wz, od, ks, build in LEBANON:
    w(f"| {need} | **{imp}** | {wz} | {od} | {ks} | {build} |")
w("")
w("## How we become their nightmare\n")
for s in KILL_SHOTS: w(f"- {s}")
w("\n## Risks & rules for ourselves\n")
for s in RISKS: w(f"- {s}")
w("")
open(os.path.join(OUT, "WIZARD_GAP_ROADMAP.md"), "w").write("\n".join(md))

# ------------------------------------------------------------------ HTML
def inline(s):
    s = html.escape(s)
    s = re.sub(r"\*\*(.+?)\*\*", r"<strong>\1</strong>", s)
    s = re.sub(r"`(.+?)`", r"<code>\1</code>", s)
    return s

H = []
h = H.append
h(f"""<!DOCTYPE html>
<html lang="en" style="color-scheme:light"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{html.escape(TITLE)}</title>
<style>
:root{{--bg:#f7f7f5;--card:#fff;--ink:#1d1f23;--muted:#62666d;--line:#e3e3df;--accent:#2f5bd3;
--ahead:#0f8a5f;--have:#3d7a3d;--partial:#b7791f;--missing:#c0392b;--skip:#8a8f98;
--p0:#c0392b;--p1:#d9822b;--p2:#2f5bd3;--p3:#8a8f98}}
*{{box-sizing:border-box}}
body{{margin:0;font:15px/1.55 -apple-system,BlinkMacSystemFont,"Segoe UI",Inter,Roboto,sans-serif;background:var(--bg);color:var(--ink)}}
header{{padding:40px 24px 24px;max-width:1180px;margin:auto}}
header h1{{font-size:30px;line-height:1.2;margin:0 0 6px}}
header .sub{{color:var(--muted)}}
main{{max-width:1180px;margin:auto;padding:0 24px 80px}}
nav.top{{position:sticky;top:0;z-index:5;background:var(--bg);border-bottom:1px solid var(--line);padding:10px 24px;display:flex;gap:16px;flex-wrap:wrap;justify-content:center}}
nav.top a{{color:var(--muted);text-decoration:none;font-size:14px}} nav.top a:hover{{color:var(--accent)}}
section{{margin-top:44px}} h2{{font-size:22px;margin:0 0 14px}} h3{{margin:0 0 6px;font-size:18px}}
.card{{background:var(--card);border:1px solid var(--line);border-radius:12px;padding:18px 20px}}
ul.tight{{margin:0;padding-left:20px}} ul.tight li{{margin:6px 0}}
.stats{{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:12px}}
.stat{{background:var(--card);border:1px solid var(--line);border-radius:12px;padding:14px 16px}}
.stat .n{{font-size:30px;font-weight:700}} .stat .l{{color:var(--muted);font-size:13px}}
.bar{{display:flex;height:12px;border-radius:6px;overflow:hidden;margin-top:14px;border:1px solid var(--line)}}
.pill{{display:inline-block;padding:1px 9px;border-radius:999px;font-size:12px;font-weight:600;color:#fff;white-space:nowrap}}
.s-ahead{{background:var(--ahead)}} .s-have{{background:var(--have)}} .s-partial{{background:var(--partial)}} .s-missing{{background:var(--missing)}} .s-skip{{background:var(--skip)}}
.p-P0{{background:var(--p0)}} .p-P1{{background:var(--p1)}} .p-P2{{background:var(--p2)}} .p-P3{{background:var(--p3)}}
.size{{border:1px solid var(--line);color:var(--muted);background:transparent}}
table{{width:100%;border-collapse:collapse;font-size:14px}}
th,td{{text-align:left;vertical-align:top;padding:8px 10px;border-bottom:1px solid var(--line)}}
th{{color:var(--muted);font-weight:600;font-size:12px;text-transform:uppercase;letter-spacing:.03em}}
td.note{{color:var(--muted)}}
.timeline{{display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:12px}}
.ms{{background:var(--card);border:1px solid var(--line);border-left:5px solid var(--p2);border-radius:10px;padding:14px 16px;text-decoration:none;color:inherit;display:block}}
.ms:hover{{border-color:var(--accent)}}
.ms.P0{{border-left-color:var(--p0)}} .ms.P1{{border-left-color:var(--p1)}} .ms.P2{{border-left-color:var(--p2)}} .ms.P3{{border-left-color:var(--p3)}}
.ms .id{{font-weight:700;color:var(--muted);font-size:13px}} .ms .t{{font-weight:650;margin:2px 0 6px}} .ms .g{{color:var(--muted);font-size:13px}}
.phase{{margin:6px 0}}
.controls{{display:flex;gap:8px;flex-wrap:wrap;align-items:center;margin:0 0 14px}}
.controls button{{border:1px solid var(--line);background:var(--card);color:var(--ink);border-radius:999px;padding:5px 12px;cursor:pointer;font-size:13px}}
.controls button.on{{background:var(--accent);color:#fff;border-color:var(--accent)}}
.controls input{{flex:1;min-width:200px;border:1px solid var(--line);background:var(--card);color:var(--ink);border-radius:8px;padding:7px 10px;font-size:14px}}
details.mod{{background:var(--card);border:1px solid var(--line);border-radius:12px;margin:10px 0;overflow:hidden}}
details.mod>summary{{cursor:pointer;padding:12px 16px;font-weight:650;display:flex;gap:10px;align-items:center;flex-wrap:wrap}}
details.mod>summary .mini{{display:flex;gap:4px;margin-left:auto}}
details.mod table{{margin:0}}
.milestone{{background:var(--card);border:1px solid var(--line);border-radius:12px;padding:20px;margin:16px 0;scroll-margin-top:60px}}
.milestone .meta{{display:flex;gap:8px;align-items:center;margin:4px 0 10px;flex-wrap:wrap}}
.milestone .why{{color:var(--muted);margin:6px 0 12px}}
.done{{margin-top:12px;padding:10px 12px;border-radius:8px;background:color-mix(in srgb,var(--ahead) 12%,transparent)}}
.edges{{display:grid;grid-template-columns:repeat(auto-fill,minmax(270px,1fr));gap:12px}}
.edge{{background:var(--card);border:1px solid var(--line);border-top:4px solid var(--ahead);border-radius:10px;padding:14px 16px}}
.edge b{{display:block;margin-bottom:4px}}
code{{background:color-mix(in srgb,var(--muted) 15%,transparent);padding:1px 5px;border-radius:4px;font-size:13px}}
.hidden{{display:none}}
figure.dia{{margin:14px 0 6px;background:#fbfbfa;border:1px solid var(--line);border-radius:12px;padding:12px 12px 6px}}
figure.dia svg{{width:100%;height:auto;display:block}}
figure.dia figcaption{{color:var(--muted);font-size:13px;padding:6px 4px 4px}}
svg.legend{{width:100%;height:auto;display:block;margin:4px 0 2px}}
.gallery{{display:grid;grid-template-columns:repeat(auto-fill,minmax(210px,1fr));gap:10px;margin-top:10px}}
.gallery a{{display:block;border:1px solid var(--line);border-radius:8px;overflow:hidden;background:#fff;text-decoration:none;color:var(--muted);font-size:12px}}
.gallery img{{width:100%;height:170px;object-fit:cover;object-position:center top;display:block;border-bottom:1px solid var(--line)}}
.gallery span{{display:block;padding:5px 8px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}}
.ghead{{margin-top:14px;font-weight:650;font-size:14px}}
#lightbox{{position:fixed;inset:0;background:rgba(10,12,16,.82);display:none;align-items:flex-start;justify-content:center;overflow:auto;z-index:50;padding:30px}}
#lightbox img{{max-width:96vw;background:#fff;border-radius:6px}}
footer{{color:var(--muted);font-size:13px;text-align:center;padding:30px}}
</style></head><body>
<nav class="top"><a href="#tldr">TL;DR</a><a href="#score">Scoreboard</a><a href="#roadmap">Roadmap</a><a href="#edges">Where we win</a>
<a href="#matrix">Feature matrix</a><a href="#milestones">Milestones</a><a href="#odoo">vs Odoo</a><a href="#lebanon">Lebanon</a><a href="#nightmare">Nightmare plan</a><a href="#risks">Risks</a></nav>
<header><h1>{html.escape(TITLE)}</h1><div class="sub">Last updated {DATE} · {total} Wizard + {ototal} Odoo capabilities compared · {len(MILESTONES)}+{len(ODOO_MILESTONES)} milestones · {nfeat + sum(len(m['features']) for m in ODOO_MILESTONES)} features to build</div></header>
<main>""")

h('<section id="tldr"><h2>TL;DR</h2><div class="card"><ul class="tight">')
for s in TLDR: h(f"<li>{inline(s)}</li>")
h('</ul></div></section>')
h('<section id="picture"><h2>The big picture</h2>' + D.legend() + D.big_picture() + '</section>')

h('<section id="score"><h2>Scoreboard</h2><div class="stats">')
for k in STATUS_LABEL:
    h(f'<div class="stat"><div class="n" style="color:var(--{k})">{counts[k]}</div><div class="l">{STATUS_ICON[k]} {html.escape(meaning[k])}</div></div>')
h('</div><div class="bar">')
for k in STATUS_LABEL:
    h(f'<div class="s-{k}" style="width:{counts[k]/total*100:.2f}%" title="{STATUS_LABEL[k]}: {counts[k]}"></div>')
h('</div><p class="sub" style="color:var(--muted);font-size:13px">Share of Wizard\'s capabilities by kashi status.</p>')
h('<h3 style="margin-top:20px">Coverage map: Wizard, module by module</h3>' + D.heatmap(MATRIX) + '</section>')

h('<section id="roadmap"><h2>Roadmap</h2><div class="card" style="margin-bottom:14px"><ul class="tight">')
for s in ORDER_NOTES: h(f'<li class="phase">{inline(s)}</li>')
h('</ul><p style="color:var(--muted);font-size:13px;margin:10px 0 0">')
h(" · ".join(f'<span class="pill p-{k}">{k}</span> {html.escape(v.split(" — ")[1])}' for k, v in PRIO_TEXT.items()))
h(' · Sizes: ' + ", ".join(f"<b>{k}</b> {v}" for k, v in SIZE_TEXT.items()) + '</p></div><div class="timeline">')
for m in MILESTONES:
    h(f'<a class="ms {m["priority"]}" href="#{m["id"].lower()}"><div class="id">{m["id"]} <span class="pill p-{m["priority"]}">{m["priority"]}</span> <span class="pill size">{m["size"]}</span></div>'
      f'<div class="t">{html.escape(m["title"])}</div><div class="g">{inline(m["goal"])}</div></a>')
h('</div></section>')

h('<section id="edges"><h2>Where kashi already wins</h2><div class="edges">')
for t, d in EDGES: h(f'<div class="edge"><b>{html.escape(t)}</b>{inline(d)}</div>')
h('</div></section>')

h('<section id="r1"><h2>What Wizard is set up with for R1 today</h2><div class="card"><table><tbody>')
for a, b in R1_SETUP: h(f'<tr><th style="width:180px">{html.escape(a)}</th><td>{inline(b)}</td></tr>')
h('</tbody></table></div></section>')

h('<section id="matrix"><h2>Full feature matrix: Wizard vs kashi</h2><div class="controls" id="mfilters"><input id="msearch" placeholder="Search features…">')
h('<button data-st="all" class="on">All</button>' + "".join(f'<button data-st="{k}">{STATUS_ICON[k]} {STATUS_LABEL[k]}</button>' for k in STATUS_LABEL))
h('<button id="expand">Expand all</button></div>')
for mod, rows in MATRIX:
    c = {k: sum(1 for r in rows if r[1] == k) for k in STATUS_LABEL}
    mini = "".join(f'<span class="pill s-{k}" title="{STATUS_LABEL[k]}">{c[k]}</span>' for k in STATUS_LABEL if c[k])
    h(f'<details class="mod"><summary>{html.escape(mod)} <span class="mini">{mini}</span></summary><table><thead><tr><th>Wizard capability</th><th style="width:120px">kashi</th><th>Notes</th></tr></thead><tbody>')
    for f, st, note in rows:
        h(f'<tr data-st="{st}"><td>{inline(f)}</td><td><span class="pill s-{st}">{STATUS_LABEL[st]}</span></td><td class="note">{inline(note)}</td></tr>')
    h('</tbody></table></details>')
h('</section>')

h('<section id="milestones"><h2>Milestones in detail</h2>' + D.legend() + '<div class="controls" id="pfilters">')
h('<button data-p="all" class="on">All priorities</button>' + "".join(f'<button data-p="{k}">{k}</button>' for k in PRIO_TEXT) + '</div>')
for m in MILESTONES:
    h(f'<div class="milestone" id="{m["id"].lower()}" data-p="{m["priority"]}"><h3>{m["id"]} · {html.escape(m["title"])}</h3>'
      f'<div class="meta"><span class="pill p-{m["priority"]}">{m["priority"]}</span><span class="pill size">{m["size"]} · {SIZE_TEXT[m["size"]]}</span></div>'
      f'<div><strong>Goal:</strong> {inline(m["goal"])}</div><div class="why">{inline(m["why"])}</div>')
    if m["id"] in D.ILLUSTRATIONS: h(D.ILLUSTRATIONS[m["id"]]())
    h('<table><thead><tr><th style="width:24%">Feature</th><th style="width:60px">Prio</th><th style="width:50px">Size</th><th>What to build</th></tr></thead><tbody>')
    for name, p, s, d in m["features"]:
        h(f'<tr data-p="{p}"><td><strong>{inline(name)}</strong></td><td><span class="pill p-{p}">{p}</span></td><td><span class="pill size">{s}</span></td><td>{inline(d)}</td></tr>')
    h('</tbody></table>')
    if m["done_when"]: h(f'<div class="done"><strong>Done when:</strong> {inline(m["done_when"])}</div>')
    pics = shots_for(m["id"])
    if pics:
        h('<div class="ghead">How Wizard does it today (their real screens)</div><div class="gallery">')
        for b in pics:
            lab = re.sub(r"^\d+_", "", b[:-4]).replace("__", " · ").replace("-", "/", 2).replace("_", " ")
            h(f'<a href="{SHOT_DIR_REL}/{b}" class="shot"><img loading="lazy" src="{SHOT_DIR_REL}/{b}" alt="{html.escape(lab)}"><span>{html.escape(lab)}</span></a>')
        h('</div>')
    h('</div>')
h('</section>')

h('<section id="odoo"><h2>Part 2 — kashi vs Odoo</h2><div class="card" style="margin-bottom:14px"><ul class="tight">')
for s_ in ODOO_INTRO: h(f"<li>{inline(s_)}</li>")
h('</ul></div><div class="stats">')
for k in STATUS_LABEL:
    h(f'<div class="stat"><div class="n" style="color:var(--{k})">{ocounts[k]}</div><div class="l">{STATUS_ICON[k]} {html.escape(meaning[k])}</div></div>')
h('</div><div class="bar">')
for k in STATUS_LABEL:
    h(f'<div class="s-{k}" style="width:{ocounts[k]/ototal*100:.2f}%" title="{STATUS_LABEL[k]}: {ocounts[k]}"></div>')
h(f'</div><p style="color:var(--muted);font-size:13px">Share of {ototal} Odoo capabilities by kashi status.</p>')
h('<h3 style="margin-top:20px">Coverage map: Odoo, app by app</h3>' + D.heatmap(ODOO_MATRIX))
for mod, rows in ODOO_MATRIX:
    c = {k: sum(1 for r in rows if r[1] == k) for k in STATUS_LABEL}
    mini = "".join(f'<span class="pill s-{k}" title="{STATUS_LABEL[k]}">{c[k]}</span>' for k in STATUS_LABEL if c[k])
    h(f'<details class="mod"><summary>Odoo · {html.escape(mod)} <span class="mini">{mini}</span></summary><table><thead><tr><th>Odoo capability</th><th style="width:120px">kashi</th><th>Notes</th></tr></thead><tbody>')
    for f, st, note in rows:
        h(f'<tr data-st="{st}"><td>{inline(f)}</td><td><span class="pill s-{st}">{STATUS_LABEL[st]}</span></td><td class="note">{inline(note)}</td></tr>')
    h('</tbody></table></details>')
h('<h3 style="margin-top:26px">Odoo-driven milestones (O-series)</h3><p style="color:var(--muted)">On top of M0–M13; Odoo features already covered by an M-milestone are not repeated.</p><div class="timeline">')
for m in ODOO_MILESTONES:
    h(f'<a class="ms {m["priority"]}" href="#{m["id"].lower()}"><div class="id">{m["id"]} <span class="pill p-{m["priority"]}">{m["priority"]}</span> <span class="pill size">{m["size"]}</span></div>'
      f'<div class="t">{html.escape(m["title"])}</div><div class="g">{inline(m["goal"])}</div></a>')
h('</div>')
for m in ODOO_MILESTONES:
    h(f'<div class="milestone" id="{m["id"].lower()}" data-p="{m["priority"]}"><h3>{m["id"]} · {html.escape(m["title"])}</h3>'
      f'<div class="meta"><span class="pill p-{m["priority"]}">{m["priority"]}</span><span class="pill size">{m["size"]} · {SIZE_TEXT[m["size"]]}</span></div>'
      f'<div><strong>Goal:</strong> {inline(m["goal"])}</div><div class="why">{inline(m["why"])}</div>')
    if m["id"] in D.ILLUSTRATIONS: h(D.ILLUSTRATIONS[m["id"]]())
    h('<table><thead><tr><th style="width:24%">Feature</th><th style="width:60px">Prio</th><th style="width:50px">Size</th><th>What to build</th></tr></thead><tbody>')
    for name, p_, sz, d in m["features"]:
        h(f'<tr data-p="{p_}"><td><strong>{inline(name)}</strong></td><td><span class="pill p-{p_}">{p_}</span></td><td><span class="pill size">{sz}</span></td><td>{inline(d)}</td></tr>')
    h('</tbody></table>')
    if m["done_when"]: h(f'<div class="done"><strong>Done when:</strong> {inline(m["done_when"])}</div>')
    pics = shots_for(m["id"])
    if pics:
        h('<div class="ghead">How Wizard does it today (their real screens)</div><div class="gallery">')
        for b in pics:
            lab = re.sub(r"^\d+_", "", b[:-4]).replace("__", " · ").replace("-", "/", 2).replace("_", " ")
            h(f'<a href="{SHOT_DIR_REL}/{b}" class="shot"><img loading="lazy" src="{SHOT_DIR_REL}/{b}" alt="{html.escape(lab)}"><span>{html.escape(lab)}</span></a>')
        h('</div>')
    h('</div>')
h('</section>')
IMP = {"Critical": "p-P0", "High": "p-P1", "High (our niche)": "p-P1", "Medium": "p-P2"}
h('<section id="lebanon"><h2>Part 3 — What matters most in the Lebanese market</h2><div class="card" style="overflow-x:auto"><table><thead><tr><th>Need</th><th>Importance</th><th>Wizard</th><th>Odoo</th><th>kashi today</th><th>What to build</th></tr></thead><tbody>')
for need, imp, wz, od, ks, build in LEBANON:
    h(f'<tr><td><strong>{inline(need)}</strong></td><td><span class="pill {IMP.get(imp,"p-P2")}">{html.escape(imp)}</span></td><td class="note">{inline(wz)}</td><td class="note">{inline(od)}</td><td>{inline(ks)}</td><td>{inline(build)}</td></tr>')
h('</tbody></table></div></section>')
h('<section id="nightmare"><h2>How we become their nightmare</h2><div class="card"><ul class="tight">')
for s in KILL_SHOTS: h(f"<li>{inline(s)}</li>")
h('</ul></div></section>')
h('<section id="risks"><h2>Risks & rules for ourselves</h2><div class="card"><ul class="tight">')
for s in RISKS: h(f"<li>{inline(s)}</li>")
h('</ul></div></section>')
h('<section id="sources"><h2>How this was researched</h2><div class="card"><ul class="tight">')
for s in SOURCES: h(f"<li>{inline(s)}</li>")
h('</ul></div></section></main><footer>kashi · internal roadmap</footer>')
h("""<div id="lightbox"><img alt=""></div><script>
const lb=document.getElementById('lightbox');document.querySelectorAll('a.shot').forEach(a=>a.onclick=e=>{e.preventDefault();lb.querySelector('img').src=a.getAttribute('href');lb.style.display='flex';});lb.onclick=()=>lb.style.display='none';
const ms=document.getElementById('msearch');let st='all';
function applyM(){const q=ms.value.toLowerCase();document.querySelectorAll('details.mod').forEach(d=>{let any=0;
d.querySelectorAll('tbody tr').forEach(r=>{const ok=(st==='all'||r.dataset.st===st)&&(!q||r.textContent.toLowerCase().includes(q));r.classList.toggle('hidden',!ok);if(ok)any++;});
d.classList.toggle('hidden',!any);if((q||st!=='all')&&any)d.open=true;});}
ms.addEventListener('input',applyM);
document.querySelectorAll('#mfilters button[data-st]').forEach(b=>b.onclick=()=>{st=b.dataset.st;document.querySelectorAll('#mfilters button[data-st]').forEach(x=>x.classList.toggle('on',x===b));applyM();});
document.getElementById('expand').onclick=e=>{const all=[...document.querySelectorAll('details.mod')];const open=!all.every(d=>d.open);all.forEach(d=>d.open=open);e.target.textContent=open?'Collapse all':'Expand all';};
document.querySelectorAll('#pfilters button').forEach(b=>b.onclick=()=>{const p=b.dataset.p;document.querySelectorAll('#pfilters button').forEach(x=>x.classList.toggle('on',x===b));
document.querySelectorAll('.milestone').forEach(m=>{let any=0;m.querySelectorAll('tbody tr').forEach(r=>{const ok=p==='all'||r.dataset.p===p;r.classList.toggle('hidden',!ok);if(ok)any++;});m.classList.toggle('hidden',!any);});});
</script></body></html>""")
open(os.path.join(OUT, "WIZARD_GAP_ROADMAP.html"), "w").write("\n".join(H))
print("matrix", counts, "total", total, "features", nfeat)
