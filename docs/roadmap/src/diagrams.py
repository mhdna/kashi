# Inline SVG illustrations for the roadmap page. Pure functions returning SVG strings.
import html

C = {
    "have": ("#e7f5ec", "#0f8a5f", "#0b5e41"),     # fill, stroke, text
    "partial": ("#fdf3e1", "#b7791f", "#7a4f0f"),
    "missing": ("#fdecea", "#c0392b", "#8e2418"),
    "neutral": ("#eef1f8", "#7d8aa8", "#2c3550"),
    "wizard": ("#e8eefc", "#2f5bd3", "#1d3b8f"),
}
FONT = "font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Inter,Roboto,sans-serif"


def _esc(s):
    return html.escape(str(s))


def _box(x, y, w, h, title, sub="", kind="neutral", dashed=None):
    fill, stroke, ink = C[kind]
    dashed = (kind == "missing") if dashed is None else dashed
    da = ' stroke-dasharray="6 4"' if dashed else ""
    out = [f'<rect x="{x}" y="{y}" width="{w}" height="{h}" rx="10" fill="{fill}" stroke="{stroke}" stroke-width="1.6"{da}/>']
    lines = title.split("\n")
    ty = y + h / 2 - (len(lines) - 1) * 8 - (7 if sub else 0) + 5
    for i, l in enumerate(lines):
        out.append(f'<text x="{x + w / 2}" y="{ty + i * 16}" text-anchor="middle" font-size="13.5" font-weight="650" fill="{ink}">{_esc(l)}</text>')
    if sub:
        for j, sl in enumerate(sub.split("\n")):
            out.append(f'<text x="{x + w / 2}" y="{ty + len(lines) * 16 + 2 + j * 14}" text-anchor="middle" font-size="11.5" fill="{ink}" opacity=".85">{_esc(sl)}</text>')
    return "".join(out)


def _arrow(x1, y1, x2, y2, label="", color="#7d8aa8", dashed=False):
    da = ' stroke-dasharray="5 4"' if dashed else ""
    out = f'<line x1="{x1}" y1="{y1}" x2="{x2}" y2="{y2}" stroke="{color}" stroke-width="1.8" marker-end="url(#ah)"{da}/>'
    if label:
        mx, my = (x1 + x2) / 2, (y1 + y2) / 2
        out += f'<text x="{mx}" y="{my - 7}" text-anchor="middle" font-size="11" fill="#555c66">{_esc(label)}</text>'
    return out


def _svg(w, h, body, caption=""):
    defs = ('<defs><marker id="ah" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">'
            '<path d="M0,0 L10,5 L0,10 z" fill="#7d8aa8"/></marker></defs>')
    cap = f'<figcaption>{caption}</figcaption>' if caption else ""
    return (f'<figure class="dia"><svg viewBox="0 0 {w} {h}" role="img" style="{FONT}" xmlns="http://www.w3.org/2000/svg">'
            f'{defs}{body}</svg>{cap}</figure>')


def legend():
    items = [("have", "kashi has it"), ("partial", "partly there"), ("missing", "to build"), ("wizard", "how Wizard/Odoo does it")]
    out, x = [], 10
    for k, t in items:
        fill, stroke, _ = C[k]
        da = ' stroke-dasharray="5 3"' if k == "missing" else ""
        out.append(f'<rect x="{x}" y="8" width="22" height="16" rx="4" fill="{fill}" stroke="{stroke}"{da}/>'
                   f'<text x="{x + 30}" y="21" font-size="12.5" fill="#333">{t}</text>')
        x += 40 + len(t) * 7
    return f'<svg viewBox="0 0 {x} 32" class="legend" style="{FONT};max-width:{x}px">{"".join(out)}</svg>'


def flow(steps, caption="", w=1100, box_w=None, h=150, arrows=None):
    """steps: list of (title, sub, kind). Horizontal chain."""
    n = len(steps)
    gap = 34
    bw = box_w or (w - 20 - gap * (n - 1)) / n
    body, y, bh = [], 30, 86
    for i, (t, s, k) in enumerate(steps):
        x = 10 + i * (bw + gap)
        body.append(_box(x, y, bw, bh, t, s, k))
        if i < n - 1:
            lab = arrows[i] if arrows else ""
            body.append(_arrow(x + bw + 2, y + bh / 2, x + bw + gap - 2, y + bh / 2, lab))
    return _svg(w, h, "".join(body), caption)


def heatmap(groups, title_w=330, cell=15, gap=3, per_row=None):
    """groups: [(name, [(feature, status, note)])] -> one row of squares per group, colored by status."""
    colors = {"ahead": "#0f8a5f", "have": "#5aa469", "partial": "#e0a43a", "missing": "#d65745", "skip": "#c9ccd2"}
    order = ["ahead", "have", "partial", "missing", "skip"]
    rows = []
    maxn = max(len(r) for _, r in groups)
    width = title_w + maxn * (cell + gap) + 60
    y = 10
    for name, items in groups:
        items = sorted(items, key=lambda r: order.index(r[1]))
        short = name if len(name) < 46 else name[:44] + "…"
        rows.append(f'<text x="{title_w - 10}" y="{y + cell - 3}" text-anchor="end" font-size="12.5" fill="#333">{_esc(short)}</text>')
        for i, (f, st, note) in enumerate(items):
            x = title_w + i * (cell + gap)
            tip = _esc(f"{f} — {st}" + (f": {note}" if note else ""))
            rows.append(f'<rect x="{x}" y="{y}" width="{cell}" height="{cell}" rx="3" fill="{colors[st]}"><title>{tip}</title></rect>')
        have = sum(1 for r in items if r[1] in ("ahead", "have"))
        rows.append(f'<text x="{title_w + len(items) * (cell + gap) + 6}" y="{y + cell - 3}" font-size="11.5" fill="#777">{have}/{len(items)}</text>')
        y += cell + 9
    lg, lx = [], title_w
    for k, t in [("ahead", "kashi ahead"), ("have", "have"), ("partial", "partial"), ("missing", "missing"), ("skip", "skip")]:
        lg.append(f'<rect x="{lx}" y="{y + 6}" width="12" height="12" rx="3" fill="{colors[k]}"/><text x="{lx + 17}" y="{y + 16}" font-size="11.5" fill="#555">{t}</text>')
        lx += 30 + len(t) * 7
    return _svg(width, y + 30, "".join(rows) + "".join(lg), "Each square is one capability; hover a square for its name and note. The number is how many kashi has (or does better).")


# ---------------------------------------------------------------------------- per-milestone illustrations

def m0():
    a = flow([
        ("Quotation", "SF · validity, terms", "missing"),
        ("Sales order", "SO · reserve, partial", "missing"),
        ("Delivery note", "DO · pick & sign", "missing"),
        ("Invoice", "SA · today: POS only", "partial"),
        ("Receipt", "RV · allocate to invoices", "missing"),
    ], arrows=["convert", "deliver", "invoice", "collect"])
    b = flow([
        ("Numbering", "SA/2026/9 per type", "missing"),
        ("Rate by date", "rate stored on doc", "missing"),
        ("KG & units", "decimal qty", "missing"),
        ("Print · Email\nWhatsApp", "A4 templates", "partial"),
        ("Audit · Attach", "who changed what", "have"),
    ], caption="Every document type gets the same engine: one chain of conversions plus shared plumbing. This is what makes Wizard feel 'complete'.")
    return a + b


def m1():
    body = []
    body.append(_box(10, 20, 220, 90, "Till 'On Account'", "debt lives only in\nthe till's SQLite", "partial"))
    body.append(_box(10, 140, 220, 90, "HQ wholesale invoice", "client price list,\ncredit check", "missing"))
    body.append(_box(330, 70, 250, 110, "Client ledger at HQ", "balance per currency\ncredit limit · terms · due dates", "missing"))
    body.append(_arrow(232, 65, 328, 110, "sync up"))
    body.append(_arrow(232, 185, 328, 145, "post"))
    outs = [("Statement of account", "PDF, USD & LBP columns", "missing"), ("Aging 0-30-60-90+", "by client & salesman", "missing"),
            ("Receipt voucher", "allocate to invoices", "missing"), ("WhatsApp reminder", "'you owe $X, statement attached'", "partial")]
    for i, (t, s, k) in enumerate(outs):
        y = 5 + i * 62
        body.append(_box(690, y, 270, 52, t, s, k))
        body.append(_arrow(582, 125, 688, y + 26))
    body.append(_box(990, 70, 100, 110, "Credit\nblock", "over limit →\nmanager", "missing"))
    body.append(_arrow(582, 125, 988, 125, "", dashed=True))
    return _svg(1100, 255, "".join(body), "Receivables: today a client's debt is stuck in one till. M1 moves it to one ledger at HQ that drives statements, aging, collections and credit blocks.")


def m2():
    body = []
    body.append(_box(10, 30, 150, 70, "Purchase order", "expected date", "missing"))
    body.append(_box(200, 30, 150, 70, "Goods receipt", "scan, count diffs", "missing"))
    body.append(_box(390, 30, 170, 70, "Purchase invoice", "today: exists", "have"))
    body.append(_arrow(162, 65, 198, 65)); body.append(_arrow(352, 65, 388, 65))
    costs = [("Freight", "$600"), ("Customs", "$900"), ("Clearing agent", "$250")]
    for i, (t, v) in enumerate(costs):
        x = 250 + i * 170
        body.append(_box(x, 150, 150, 56, t, v, "missing"))
        body.append(_arrow(x + 75, 150, 610, 100, "", dashed=True))
    body.append(_box(600, 30, 200, 70, "Landed cost", "split by value / kg / qty", "missing"))
    body.append(_arrow(562, 65, 598, 65))
    body.append(_box(840, 10, 240, 52, "True unit cost", "$4.10 → $4.85 per piece", "missing"))
    body.append(_box(840, 78, 240, 52, "Selling prices", "cost × markup per price list", "missing"))
    body.append(_box(840, 146, 240, 52, "Supplier balance", "pay · return · statement", "missing"))
    for y in (36, 104, 172):
        body.append(_arrow(802, 65, 838, y))
    return _svg(1100, 225, "".join(body), "Purchasing: the margin is decided here. Without landed cost, a container's freight and customs never reach the item's cost or its price.")


def m3():
    body = []
    body.append(_box(10, 70, 200, 110, "Bale 50 kg", "landed cost $410\n(from M2)", "neutral"))
    body.append(_box(260, 85, 170, 80, "Opening /\nsorting", "who sorted, when", "missing"))
    body.append(_arrow(212, 125, 258, 125))
    grades = [("Crème", "38 pcs · $5.20 each", "missing"), ("Extra", "61 pcs · $2.70 each", "missing"),
              ("L1", "44 pcs · $1.10 each", "missing"), ("Waste", "3.5 kg · $0", "missing")]
    for i, (t, s, k) in enumerate(grades):
        y = 5 + i * 62
        body.append(_box(500, y, 230, 52, t, s, k))
        body.append(_arrow(432, 125, 498, y + 26))
    body.append(_box(790, 40, 290, 70, "Yield by supplier", "China bales: 22% Crème · Turkey: 31%", "missing"))
    body.append(_box(790, 140, 290, 70, "Cost per graded piece", "split by weight, ratio or expected value", "missing"))
    body.append(_arrow(732, 125, 788, 75)); body.append(_arrow(732, 125, 788, 175))
    return _svg(1100, 255, "".join(body), "Bale opening (our niche): one bale becomes graded stock with a real cost per piece, and we learn which supplier's bales yield the best grades. Figures are illustrative.")


def m4():
    return flow([
        ("Quote", "valid 15 days", "missing"),
        ("Order", "reserves stock", "missing"),
        ("Preparing", "picker assigned", "missing"),
        ("Delivered 60%", "delivery note 1", "missing"),
        ("Invoiced 60%", "rest stays open", "missing"),
        ("Delivered 100%", "note 2 → invoice 2", "missing"),
    ], caption="Partial fulfilment: one order, two deliveries, two invoices, and the order always knows what's still open.")


def m5():
    body = []
    docs = [("Sales invoice", "have"), ("Purchase invoice", "have"), ("Expense", "have"), ("Shift close", "have"), ("Receipt / payment", "missing"), ("Payroll", "missing")]
    for i, (t, k) in enumerate(docs):
        y = 5 + i * 44
        body.append(_box(10, y, 190, 36, t, "", k))
        body.append(_arrow(202, y + 18, 278, 135))
    body.append(_box(280, 85, 190, 100, "Posting rules", "per document type\n(written once)", "missing"))
    body.append(_box(530, 85, 220, 100, "Journal", "Dr 411 Clients  $668\nCr 701 Sales     $668", "missing"))
    body.append(_arrow(472, 135, 528, 135))
    outs = ["Trial balance", "Profit & loss", "Balance sheet", "Cash flow", "Account statements"]
    for i, t in enumerate(outs):
        y = 5 + i * 52
        body.append(_box(820, y, 260, 42, t, "", "missing"))
        body.append(_arrow(752, 135, 818, y + 21))
    return _svg(1100, 275, "".join(body), "The ledger: the documents we already have (green) start posting journal entries automatically, and the accountant gets real statements.")


def m6():
    a = flow([
        ("Cheque received", "from client, due 30/11", "missing"),
        ("In the safe", "post-dated register", "missing"),
        ("Deposited", "bank account", "missing"),
        ("Cleared ✓", "client paid", "missing"),
    ], arrows=["", "on due date", "bank confirms"], caption="")
    b = flow([
        ("Bounced ✗", "back to client debt", "missing"),
        ("VAT on every line", "tax code per product", "missing"),
        ("VAT return", "collected − deductible", "missing"),
    ], caption="Banks, cheques & VAT: the cheque lifecycle (with the bounce path) and VAT that comes out of the same documents.")
    return a + b


def fx():
    body = []
    body.append(_box(10, 40, 240, 90, "Invoice 01/10", "$1,000 @ 89,500\n= 89.5M LBP", "missing"))
    body.append(_box(330, 10, 250, 70, "Paid $600 cash", "01/11", "missing"))
    body.append(_box(330, 100, 250, 70, "Paid 36M LBP", "01/11 @ 90,000 = $400", "missing"))
    body.append(_arrow(252, 85, 328, 45)); body.append(_arrow(252, 85, 328, 135))
    body.append(_box(660, 40, 220, 90, "Balance $0", "rate difference\n→ FX gain/loss", "missing"))
    body.append(_arrow(582, 45, 658, 75)); body.append(_arrow(582, 135, 658, 100))
    body.append(_box(940, 40, 150, 90, "Rate table", "one rate per day,\nkept forever", "missing"))
    return _svg(1100, 180, "".join(body), "Dual currency done right: every document keeps its own rate, partial payments in different currencies close the invoice, and the difference is booked. Rates are illustrative.")


def m7():
    return flow([
        ("Cashier gives 25% off", "on a $300 sale", "have"),
        ("Rule matches", "discount > 15%", "missing"),
        ("Waits for approval", "manager notified", "partial"),
        ("Approve on WhatsApp", "one tap, logged", "missing"),
        ("Sale continues", "audit trail", "have"),
    ], caption="Approvals: one engine for every document (discounts, below-cost, over credit limit, big purchases), approvable from the web, the till or WhatsApp.")


def m8():
    return flow([
        ("Door punches", "Akuvox device", "have"),
        ("Rota (plan)", "who should be in", "missing"),
        ("Late / overtime", "computed, not typed", "missing"),
        ("Payroll run", "+commission −advances", "missing"),
        ("Payslip", "sent on WhatsApp", "missing"),
        ("NSSF · tax · EOS", "Lebanese reports", "missing"),
    ], caption="Payroll built on the attendance we already capture: plan vs actual becomes deductions automatically.")


def m9():
    return flow([
        ("Sign up", "company, branches", "missing"),
        ("Import from Wizard", "clients, products,\nbalances, COA", "missing"),
        ("Arabic UI", "RTL, documents", "partial"),
        ("Modules on/off", "per plan", "missing"),
        ("Selling day 1", "till + HQ", "have"),
    ], caption="Productizing: the path a new customer walks; the Wizard import is our sharpest weapon.")


def m10():
    return flow([
        ("Load van", "transfer to vehicle", "missing"),
        ("Visit client", "GPS check-in", "missing"),
        ("Order / sell", "client price & credit", "missing"),
        ("Collect cash", "receipt on WhatsApp", "missing"),
        ("Offline all day", "syncs later", "partial"),
        ("Settle van", "cash & stock back", "missing"),
    ], caption="A salesman's day in the mobile app; offline sync is the part we already know how to do.")


def m11():
    return flow([
        ("WhatsApp message", "'do you have…'", "have"),
        ("Lead", "stage, owner", "missing"),
        ("Quotation", "sent as PDF", "missing"),
        ("Client", "first invoice", "have"),
        ("Portal link", "points, balance,\ninvoices", "missing"),
        ("Campaigns", "segments, journeys", "partial"),
    ], caption="CRM on WhatsApp: the conversation we already have becomes the pipeline.")


def m12():
    return flow([
        ("kashi product", "variants, photos", "have"),
        ("Shopify / Woo", "push stock & price", "missing"),
        ("Web order", "COD", "missing"),
        ("Reserve stock", "same number as shops", "missing"),
        ("Courier & COD", "reconcile cash", "missing"),
    ], caption="One stock number for shops and online.")


def o1():
    return flow([
        ("Photo of a bill", "already collected today", "have"),
        ("AI reads it", "supplier, lines, VAT", "missing"),
        ("You confirm", "one tap", "missing"),
        ("Posted", "expense / purchase", "missing"),
        ("Ask kashi", "'who owes > $500?'", "missing"),
    ], caption="AI layer: Odoo sells this as paid credits; Wizard doesn't have it.")


def o2():
    return flow([
        ("WhatsApp link", "OTP login", "missing"),
        ("See balance", "statement, invoices", "missing"),
        ("Sign quote", "signature pad", "missing"),
        ("Pay", "Whish / card link", "missing"),
    ], caption="Customer portal, WhatsApp-first.")


def o3():
    return flow([
        ("Scale", "weight at checkout", "missing"),
        ("Weighed barcode", "price inside EAN", "missing"),
        ("Gift card / eWallet", "refund to credit", "missing"),
        ("Cash drawer kick", "opens on cash", "missing"),
        ("Card terminal", "amount sent", "missing"),
    ], caption="Till hardware parity with Odoo POS.")


def o6():
    return flow([
        ("Footfall by hour", "visitor counter", "have"),
        ("Suggested rota", "staff where busy", "missing"),
        ("Published", "WhatsApp to staff", "missing"),
        ("Punches vs plan", "late, absent", "missing"),
        ("Payroll", "M8", "missing"),
    ], caption="Staff planning that only kashi can do: rota from real footfall.")


ILLUSTRATIONS = {"M0": m0, "M1": lambda: m1() + fx(), "M2": m2, "M3": m3, "M4": m4, "M5": m5, "M6": m6, "M7": m7, "M8": m8,
                 "M9": m9, "M10": m10, "M11": m11, "M12": m12, "O1": o1, "O2": o2, "O3": o3, "O6": o6}


def big_picture():
    """Shop floor vs back office map: where each product is strong."""
    body = []
    body.append('<text x="275" y="22" text-anchor="middle" font-size="15" font-weight="700" fill="#0b5e41">Shop floor &amp; branches</text>')
    body.append('<text x="825" y="22" text-anchor="middle" font-size="15" font-weight="700" fill="#8e2418">Back office &amp; accounting</text>')
    left = [("Offline till", "have"), ("Shifts & cash", "have"), ("Promotions", "have"), ("Loyalty", "have"), ("WhatsApp", "have"),
            ("Attendance", "have"), ("Transfers", "have"), ("Analytics", "have"), ("Footfall", "have"), ("Remote control", "have"),
            ("Stock counts", "partial"), ("Barcodes", "have")]
    right = [("General ledger", "missing"), ("Receivables", "missing"), ("Payables", "missing"), ("Quote→Order", "missing"),
             ("Purchase orders", "missing"), ("Landed cost", "missing"), ("Banks & cheques", "missing"), ("VAT", "missing"),
             ("Payroll", "missing"), ("Units / KG", "missing"), ("Approvals", "partial"), ("Fixed assets", "partial")]
    for side, items, x0 in (("l", left, 20), ("r", right, 570)):
        for i, (t, k) in enumerate(items):
            col, row = i % 3, i // 3
            body.append(_box(x0 + col * 175, 40 + row * 56, 162, 44, t, "", k))
    body.append('<line x1="550" y1="34" x2="550" y2="268" stroke="#c9ccd2" stroke-width="2" stroke-dasharray="4 5"/>')
    return _svg(1100, 280, "".join(body), "Where kashi stands today: strong on the shop floor (left), mostly missing in the back office (right) — which is exactly where Wizard and Odoo are strong.")
