"""
OpsOwl AI × Decentro — PowerPoint generator
Run: python3 gen_pptx.py
Output: OpsOwl-Decentro-Pitch.pptx
"""

from pptx import Presentation
from pptx.util import Inches, Pt, Emu
from pptx.dml.color import RGBColor
from pptx.enum.text import PP_ALIGN
from pptx.util import Inches, Pt
import pptx.util as util

# ── Palette ──────────────────────────────────────────────────────────────────
BG      = RGBColor(0x06, 0x0e, 0x1a)   # #060e1a
SURF    = RGBColor(0x0d, 0x1b, 0x2e)   # #0d1b2e
SURF2   = RGBColor(0x11, 0x1f, 0x33)
BORDER  = RGBColor(0x1a, 0x2f, 0x4a)
CYAN    = RGBColor(0x00, 0xc8, 0xe0)
AMBER   = RGBColor(0xf5, 0x9e, 0x0b)
GREEN   = RGBColor(0x10, 0xb9, 0x81)
RED     = RGBColor(0xf8, 0x71, 0x71)
WHITE   = RGBColor(0xff, 0xff, 0xff)
SUBTLE  = RGBColor(0x94, 0xa3, 0xb8)
MUTED   = RGBColor(0x4a, 0x60, 0x80)

W = Inches(13.333)   # widescreen 16:9
H = Inches(7.5)


def new_prs():
    prs = Presentation()
    prs.slide_width  = W
    prs.slide_height = H
    return prs


def blank_slide(prs):
    layout = prs.slide_layouts[6]   # completely blank
    return prs.slides.add_slide(layout)


def fill_bg(slide, color=BG):
    bg = slide.background
    fill = bg.fill
    fill.solid()
    fill.fore_color.rgb = color


def box(slide, left, top, width, height, bg=None, border=None, border_w=Pt(0.5)):
    """Add a filled rectangle (no text)."""
    shape = slide.shapes.add_shape(
        1,   # MSO_SHAPE_TYPE.RECTANGLE
        left, top, width, height
    )
    shape.line.width = 0
    if bg:
        shape.fill.solid()
        shape.fill.fore_color.rgb = bg
    else:
        shape.fill.background()
    if border:
        shape.line.color.rgb = border
        shape.line.width = border_w
    else:
        shape.line.fill.background()
    return shape


def txbox(slide, text, left, top, width, height,
          font_size=Pt(12), font_color=WHITE, bold=False,
          align=PP_ALIGN.LEFT, wrap=True, italic=False,
          bg=None, border=None):
    """Add a text box."""
    tb = slide.shapes.add_textbox(left, top, width, height)
    tf = tb.text_frame
    tf.word_wrap = wrap
    p = tf.paragraphs[0]
    p.alignment = align
    run = p.add_run()
    run.text = text
    run.font.size = font_size
    run.font.color.rgb = font_color
    run.font.bold = bold
    run.font.italic = italic
    run.font.name = "Segoe UI"
    return tb


def header(slide, slide_num_text, total=8):
    """Render the shared top header bar."""
    # Header bar background
    box(slide, 0, 0, W, Inches(0.55), bg=SURF, border=BORDER)
    # Logo mark
    box(slide, Inches(0.4), Inches(0.1), Inches(0.35), Inches(0.35), bg=CYAN)
    txbox(slide, "O", Inches(0.4), Inches(0.1), Inches(0.35), Inches(0.35),
          font_size=Pt(13), font_color=WHITE, bold=True, align=PP_ALIGN.CENTER)
    # Logo name
    txbox(slide, "OpsOwl AI", Inches(0.82), Inches(0.1), Inches(1.5), Inches(0.22),
          font_size=Pt(10), font_color=WHITE, bold=True)
    txbox(slide, "FINCRIME INVESTIGATIONS & OPERATIONS", Inches(0.82), Inches(0.3), Inches(2), Inches(0.18),
          font_size=Pt(6.5), font_color=MUTED)
    # Slide number (right aligned)
    txbox(slide, slide_num_text, Inches(10.5), Inches(0.17), Inches(2.4), Inches(0.22),
          font_size=Pt(8), font_color=MUTED, align=PP_ALIGN.RIGHT)


def divider(slide, left, top, width=Inches(0.5)):
    box(slide, left, top, width, Pt(3), bg=CYAN)


def add_para(tf, text, font_size=Pt(11), color=SUBTLE, bold=False, space_before=Pt(4), italic=False):
    p = tf.add_paragraph()
    p.space_before = space_before
    r = p.add_run()
    r.text = text
    r.font.size = font_size
    r.font.color.rgb = color
    r.font.bold = bold
    r.font.italic = italic
    r.font.name = "Segoe UI"
    return p


def card_box(slide, left, top, width, height, accent_color=None, title=None, lines=None, font_size=Pt(10.5)):
    """Render a card with optional title and bullet lines."""
    bg_col = SURF
    bdr_col = BORDER
    if accent_color:
        bdr_col = accent_color
    box(slide, left, top, width, height, bg=bg_col, border=bdr_col)

    y = top + Inches(0.15)
    if title:
        txbox(slide, title, left + Inches(0.18), y, width - Inches(0.36), Inches(0.24),
              font_size=Pt(11), font_color=WHITE, bold=True)
        y += Inches(0.26)
    if lines:
        for line in lines:
            col = SUBTLE
            if line.startswith("✓") or line.startswith("✅"):
                col = GREEN
            elif line.startswith("✗") or line.startswith("❌"):
                col = RED
            txbox(slide, line, left + Inches(0.18), y, width - Inches(0.36), Inches(0.22),
                  font_size=font_size, font_color=col)
            y += Inches(0.22)


# ══════════════════════════════════════════════════════════════════════════════
# SLIDE 1 — HERO
# ══════════════════════════════════════════════════════════════════════════════
def slide1(prs):
    sl = blank_slide(prs)
    fill_bg(sl)
    header(sl, "CONFIDENTIAL · FOR DECENTRO")

    # Eyebrow
    txbox(sl, "ENTERPRISE PRESENTATION · 2026",
          Inches(0.6), Inches(1.0), Inches(7), Inches(0.25),
          font_size=Pt(9), font_color=CYAN, bold=True)
    divider(sl, Inches(0.6), Inches(1.3))

    # Main title
    tb = sl.shapes.add_textbox(Inches(0.6), Inches(1.5), Inches(9), Inches(2.4))
    tf = tb.text_frame
    tf.word_wrap = True

    p = tf.paragraphs[0]
    p.add_run().text = "The Investigation"
    p.runs[0].font.size = Pt(44)
    p.runs[0].font.color.rgb = WHITE
    p.runs[0].font.bold = True
    p.runs[0].font.name = "Segoe UI"

    add_para(tf, "Operations Layer", Pt(44), WHITE, bold=True)
    # last line in cyan
    p3 = tf.add_paragraph()
    r = p3.add_run()
    r.text = "for India's Fintech Stack."
    r.font.size = Pt(44)
    r.font.color.rgb = CYAN
    r.font.bold = True
    r.font.name = "Segoe UI"

    # Description
    txbox(sl,
          "OpsOwl AI is the Fincrime Investigations & Operations Workspace — the system of record "
          "that sits between a TMS alert and a filed STR, purpose-built for the full AML investigation lifecycle.",
          Inches(0.6), Inches(4.0), Inches(7.5), Inches(0.7),
          font_size=Pt(14), font_color=SUBTLE)

    # Tags
    tags = [
        ("On-Premise AI", CYAN),
        ("FIU-IND Native", CYAN),
        ("goAML 4.0 Export", CYAN),
        ("4-Eye Maker-Checker", AMBER),
        ("Hash-Chain Audit Trail", GREEN),
    ]
    x = Inches(0.6)
    for tag, col in tags:
        w = Inches(1.8)
        b = box(sl, x, Inches(4.85), w, Inches(0.32), bg=SURF, border=col)
        txbox(sl, tag, x, Inches(4.85), w, Inches(0.32),
              font_size=Pt(9), font_color=col, bold=True, align=PP_ALIGN.CENTER)
        x += w + Inches(0.1)


# ══════════════════════════════════════════════════════════════════════════════
# SLIDE 2 — THE PROBLEM
# ══════════════════════════════════════════════════════════════════════════════
def slide2(prs):
    sl = blank_slide(prs)
    fill_bg(sl)
    header(sl, "02 / 08")

    txbox(sl, "The Industry Solved Detection.",
          Inches(0.5), Inches(0.65), Inches(9), Inches(0.5),
          font_size=Pt(28), font_color=WHITE, bold=True)
    txbox(sl, "Investigation Is Still Broken.",
          Inches(0.5), Inches(1.1), Inches(9), Inches(0.5),
          font_size=Pt(28), font_color=CYAN, bold=True)
    txbox(sl, "The investigation workflow hasn't changed in 15 years, while detection became a machine.",
          Inches(0.5), Inches(1.58), Inches(9), Inches(0.3),
          font_size=Pt(11), font_color=SUBTLE)

    # Left card — BEFORE (automated)
    box(sl, Inches(0.5), Inches(2.0), Inches(4.8), Inches(2.0), bg=SURF, border=GREEN)
    txbox(sl, "BEFORE ALERT — FULLY AUTOMATED",
          Inches(0.68), Inches(2.08), Inches(4.4), Inches(0.2),
          font_size=Pt(7.5), font_color=GREEN, bold=True)
    txbox(sl, "Transaction Monitoring",
          Inches(0.68), Inches(2.28), Inches(4.4), Inches(0.22),
          font_size=Pt(11), font_color=WHITE, bold=True)
    before_items = [
        "✓  Real-time rule engines & ML scoring",
        "✓  Instant alert generation on threshold breach",
        "✓  Zero manual steps — fully automated pipeline",
    ]
    y = 2.52
    for item in before_items:
        txbox(sl, item, Inches(0.68), Inches(y), Inches(4.4), Inches(0.24),
              font_size=Pt(10), font_color=SUBTLE)
        y += 0.24

    # Right card — AFTER (manual)
    box(sl, Inches(5.5), Inches(2.0), Inches(5.3), Inches(3.8), bg=SURF, border=RED)
    txbox(sl, "AFTER ALERT — STILL ENTIRELY MANUAL",
          Inches(5.68), Inches(2.08), Inches(4.9), Inches(0.2),
          font_size=Pt(7.5), font_color=RED, bold=True)
    txbox(sl, "Investigation Operations",
          Inches(5.68), Inches(2.28), Inches(4.9), Inches(0.22),
          font_size=Pt(11), font_color=WHITE, bold=True)
    after_items = [
        "✗  Analyst opens Excel — copies transaction data by hand",
        "✗  Chases KYC team over email for documents",
        '✗  STR narrative in Word — "STR_v6_FINAL_final.docx"',
        "✗  Approval by reply-all email — no audit trail",
        "✗  5–8 days average: alert to STR filing",
        "✗  RBI examination = 2–3 weeks of document scrambling",
    ]
    y = 2.52
    for item in after_items:
        txbox(sl, item, Inches(5.68), Inches(y), Inches(4.9), Inches(0.26),
              font_size=Pt(10), font_color=SUBTLE)
        y += 0.26

    # 3 stat boxes on left bottom
    stats = [("5–8", "days per investigation"), ("3–4h", "to write one STR narrative"), ("0", "purpose-built tools in India")]
    x = 0.5
    for num, lbl in stats:
        box(sl, Inches(x), Inches(4.1), Inches(1.55), Inches(0.9), bg=SURF, border=BORDER)
        txbox(sl, num, Inches(x), Inches(4.15), Inches(1.55), Inches(0.45),
              font_size=Pt(22), font_color=CYAN, bold=True, align=PP_ALIGN.CENTER)
        txbox(sl, lbl, Inches(x), Inches(4.55), Inches(1.55), Inches(0.3),
              font_size=Pt(8), font_color=MUTED, align=PP_ALIGN.CENTER)
        x += 1.65

    # Quote
    box(sl, Inches(0.5), Inches(5.15), Inches(12.3), Inches(0.75), bg=SURF, border=CYAN)
    txbox(sl,
          "Every fintech has a sophisticated TMS. When it fires, the analyst gets an email notification "
          "and opens a spreadsheet. That gap — between alert and STR — is where OpsOwl lives.",
          Inches(0.68), Inches(5.25), Inches(12.0), Inches(0.6),
          font_size=Pt(11), font_color=SUBTLE, italic=True)


# ══════════════════════════════════════════════════════════════════════════════
# SLIDE 3 — WHERE WE FIT
# ══════════════════════════════════════════════════════════════════════════════
def slide3(prs):
    sl = blank_slide(prs)
    fill_bg(sl)
    header(sl, "03 / 08")

    txbox(sl, "Where OpsOwl Fits in Your Stack",
          Inches(0.5), Inches(0.65), Inches(10), Inches(0.45),
          font_size=Pt(28), font_color=WHITE, bold=True)
    txbox(sl, "Not a TMS. Not a screening tool. The investigation layer — downstream of every alert Decentro generates.",
          Inches(0.5), Inches(1.12), Inches(11), Inches(0.28),
          font_size=Pt(11), font_color=SUBTLE)

    # Pipeline column — OpsOwl covers 3 boxes
    pipeline_pre = [
        ("KYC / Onboarding", MUTED, False),
        ("↓", MUTED, False),
        ("Sanctions & PEP Screening", MUTED, False),
        ("↓", MUTED, False),
        ("Transaction Monitoring\n(Decentro AML APIs · Client TMS)", RGBColor(0x7d,0xd3,0xfc), False),
        ("🚨  ALERT", AMBER, False),
    ]
    y = 1.5
    for text, col, is_hl in pipeline_pre:
        h = 0.35 if "\n" in text else 0.28
        txbox(sl, text, Inches(0.5), Inches(y), Inches(2.4), Inches(h),
              font_size=Pt(9), font_color=col, align=PP_ALIGN.CENTER)
        y += h + 0.02

    # OpsOwl bracket label
    txbox(sl, "OpsOwl AI covers all three", Inches(0.5), Inches(y), Inches(2.4), Inches(0.18),
          font_size=Pt(7), font_color=CYAN, bold=True, align=PP_ALIGN.CENTER)
    y += 0.2

    # 3 highlighted OpsOwl boxes
    opsowl_boxes = [
        ("Investigation Operations\nCase mgmt · Findings · AI Copilot",),
        ("STR · FIU-IND · goAML XML\nAI narrative · Maker-checker · XML export",),
        ("Regulatory Audit · Case Closure\nHash-chain audit · One-click export",),
    ]
    for (text,) in opsowl_boxes:
        box(sl, Inches(0.5), Inches(y), Inches(2.4), Inches(0.5), bg=SURF, border=CYAN)
        txbox(sl, text, Inches(0.5), Inches(y + 0.02), Inches(2.4), Inches(0.46),
              font_size=Pt(8.5), font_color=CYAN, bold=False, align=PP_ALIGN.CENTER)
        y += 0.54
        if text != opsowl_boxes[-1][0]:
            txbox(sl, "↓", Inches(0.5), Inches(y - 0.06), Inches(2.4), Inches(0.16),
                  font_size=Pt(10), font_color=MUTED, align=PP_ALIGN.CENTER)

    # Opportunity A
    box(sl, Inches(3.2), Inches(1.5), Inches(4.7), Inches(2.1), bg=SURF, border=CYAN)
    txbox(sl, "OPPORTUNITY A — DECENTRO AS CUSTOMER",
          Inches(3.38), Inches(1.58), Inches(4.3), Inches(0.2),
          font_size=Pt(7.5), font_color=CYAN, bold=True)
    txbox(sl, "PSP Compliance Operations",
          Inches(3.38), Inches(1.8), Inches(4.3), Inches(0.22),
          font_size=Pt(11), font_color=WHITE, bold=True)
    txbox(sl,
          "Decentro holds a PSP licence with direct AML obligations. Every suspicious "
          "transaction on Decentro's own platform requires investigation, documentation, "
          "and STR filing to FIU-IND. OpsOwl replaces the spreadsheet-and-email workflow "
          "for Decentro's compliance team — deployed on-premise in one day.",
          Inches(3.38), Inches(2.05), Inches(4.3), Inches(1.35),
          font_size=Pt(10), font_color=SUBTLE)

    # Opportunity B
    box(sl, Inches(8.1), Inches(1.5), Inches(4.7), Inches(2.1), bg=SURF, border=AMBER)
    txbox(sl, "OPPORTUNITY B — DECENTRO AS DISTRIBUTION",
          Inches(8.28), Inches(1.58), Inches(4.3), Inches(0.2),
          font_size=Pt(7.5), font_color=AMBER, bold=True)
    txbox(sl, "Fintech Client Investigation Layer",
          Inches(8.28), Inches(1.8), Inches(4.3), Inches(0.22),
          font_size=Pt(11), font_color=WHITE, bold=True)
    txbox(sl,
          "Decentro generates alerts for fintech clients across India via its AML APIs. "
          "Those clients receive the alert — but have no purpose-built tool to investigate it. "
          "OpsOwl becomes the investigation layer Decentro offers downstream — "
          "white-labelled, embedded, or via referral partnership.",
          Inches(8.28), Inches(2.05), Inches(4.3), Inches(1.35),
          font_size=Pt(10), font_color=SUBTLE)

    # Integration card
    box(sl, Inches(3.2), Inches(3.75), Inches(9.6), Inches(1.4), bg=SURF, border=BORDER)
    txbox(sl, "INTEGRATION — HOW IT CONNECTS",
          Inches(3.38), Inches(3.83), Inches(9.0), Inches(0.2),
          font_size=Pt(7.5), font_color=MUTED, bold=True)
    txbox(sl,
          "Decentro API fires alert  →  HMAC-signed webhook hits OpsOwl  →  Case auto-created, "
          "analyst assigned, SLA clock starts  →  Investigation completed in OpsOwl  →  STR filed  "
          "→  Decentro receives case-closed event via outbound webhook.\n"
          "Setup time: 1 day.  Any JSON schema.  Field-mapped in the UI — no code changes.",
          Inches(3.38), Inches(4.06), Inches(9.0), Inches(0.9),
          font_size=Pt(10), font_color=SUBTLE)


# ══════════════════════════════════════════════════════════════════════════════
# SLIDE 4 — INVESTIGATION LIFECYCLE
# ══════════════════════════════════════════════════════════════════════════════
def slide4(prs):
    sl = blank_slide(prs)
    fill_bg(sl)
    header(sl, "04 / 08")

    txbox(sl, "One Lifecycle. One System of Record.",
          Inches(0.5), Inches(0.65), Inches(10), Inches(0.42),
          font_size=Pt(26), font_color=WHITE, bold=True)
    txbox(sl, "Alert to case closure — every step tracked, maker-checked, and hash-chained.",
          Inches(0.5), Inches(1.1), Inches(10), Inches(0.28),
          font_size=Pt(11), font_color=SUBTLE)

    # 6 flow boxes
    steps = [
        ("🚨", "Alert In", "HMAC webhook · Any TMS · Bulk triage · 1-click case", False),
        ("📋", "Case Created", "SLA starts · Playbook tasks auto-generated · Auto-assign rules", True),
        ("🔍", "Investigate", "Docs + OCR · AI risk extraction · Transaction flow · AI Copilot", False),
        ("🛡️", "Document", "Findings + evidence · Severity · Submit → Approve · 4-eye enforced", True),
        ("📄", "Report", "AI narrative grounded in case data · STR draft + maker-checker · goAML XML", False),
        ("🔒", "Close", "Checklist gate · Closure approved · Hash-chain sealed · Audit export", True),
    ]
    step_w = Inches(2.0)
    x = Inches(0.3)
    for icon, title, body, hl in steps:
        bdr = CYAN if hl else BORDER
        box(sl, x, Inches(1.5), step_w, Inches(1.4), bg=SURF, border=bdr)
        txbox(sl, icon, x, Inches(1.55), step_w, Inches(0.3), font_size=Pt(18), align=PP_ALIGN.CENTER)
        txbox(sl, title, x, Inches(1.88), step_w, Inches(0.25),
              font_size=Pt(10), font_color=CYAN if hl else WHITE, bold=True, align=PP_ALIGN.CENTER)
        txbox(sl, body, x, Inches(2.14), step_w - Inches(0.2), Inches(0.7),
              font_size=Pt(8.5), font_color=SUBTLE, align=PP_ALIGN.CENTER)
        # arrow
        if x + step_w < Inches(12.6):
            txbox(sl, "→", x + step_w, Inches(1.95), Inches(0.2), Inches(0.3),
                  font_size=Pt(14), font_color=MUTED, align=PP_ALIGN.CENTER)
        x += step_w + Inches(0.2)

    # 4 feature cards
    feat_cards = [
        ("Playbooks", CYAN, "Zero analyst guesswork. Case type fires → tasks auto-generated. Every analyst, every time. Institutional knowledge survives staff turnover."),
        ("4-Eye Maker-Checker", AMBER, "Findings, narratives, STRs, case closure — creator is blocked from approving their own work. A second named approver is mandatory. Enforced in code."),
        ("AI Narrative", GREEN, "Reads approved findings, uploaded documents, subject profile, and AI-extracted risk factors. Does not invent amounts, occupations, or dates."),
        ("Hash-Chain Audit", BORDER, "Every action — AI or human — is hash-chained. Modified records break the chain. One-click export per case. No scrambling before an RBI examination."),
    ]
    card_w = Inches(3.1)
    x = Inches(0.4)
    for title, col, body in feat_cards:
        box(sl, x, Inches(3.1), card_w, Inches(1.95), bg=SURF, border=col)
        txbox(sl, title, x + Inches(0.15), Inches(3.18), card_w - Inches(0.3), Inches(0.25),
              font_size=Pt(11), font_color=WHITE, bold=True)
        txbox(sl, body, x + Inches(0.15), Inches(3.46), card_w - Inches(0.3), Inches(1.5),
              font_size=Pt(9.5), font_color=SUBTLE)
        x += card_w + Inches(0.17)


# ══════════════════════════════════════════════════════════════════════════════
# SLIDE 5 — LIVE DEMO
# ══════════════════════════════════════════════════════════════════════════════
def slide5(prs):
    sl = blank_slide(prs)
    fill_bg(sl)
    header(sl, "05 / 08")

    txbox(sl, "Live Product Walkthrough",
          Inches(0.5), Inches(0.65), Inches(10), Inches(0.42),
          font_size=Pt(26), font_color=WHITE, bold=True)
    txbox(sl, "One investigation. Alert to STR. Running on-premise. Unedited.",
          Inches(0.5), Inches(1.1), Inches(10), Inches(0.28),
          font_size=Pt(11), font_color=SUBTLE)

    # Timeline
    timeline = [
        ("DAY 1 · 14:23 IST", "Alert fires. Velocity breach — Rajesh Kumar Gupta, Axis Bank. 14 UPI transfers, ₹22.6L in 6 hours. Arrives via TMS webhook to OpsOwl."),
        ("DAY 1 · 14:24 IST", "Case OWL-2026-00005 created. SLA: 7 days. Playbook auto-generates 6 investigation tasks. Assigned to Priya Sharma (Analyst)."),
        ("DAY 1 · 15:00 IST", "KYC doc + bank statement uploaded. On-premise AI extracts: risk score 85/100 · 5 risk factors · 4 transaction anomalies. In 45 seconds."),
        ("DAY 2 · 10:00 IST", "3 findings documented with evidence. Structuring HIGH · Layering HIGH · KYC Re-rating MEDIUM. Submitted by analyst. 4-eye rule blocks self-approval."),
        ("DAY 2 · 11:00 IST", "Manager approves findings. AI narrative generated — 7 paragraphs, grounded in uploaded documents and approved findings. Reviewed and approved."),
        ("DAY 2 · 14:00 IST", "STR filed. Ref: FIU-IND/STR/2024-25/MH/087341. goAML XML exported. 25 hours total. Zero emails. Zero Excel. Zero Word documents."),
    ]
    y = 1.5
    dot_x = Inches(0.65)
    line_x = Inches(0.72)
    for i, (time_str, body) in enumerate(timeline):
        # dot
        box(sl, dot_x, Inches(y + 0.04), Inches(0.12), Inches(0.12), bg=CYAN)
        txbox(sl, time_str, Inches(0.85), Inches(y), Inches(2.5), Inches(0.22),
              font_size=Pt(8.5), font_color=CYAN, bold=True)
        txbox(sl, body, Inches(0.85), Inches(y + 0.22), Inches(5.8), Inches(0.4),
              font_size=Pt(9.5), font_color=SUBTLE)
        y += 0.76

    # Right panel — What you'll see
    box(sl, Inches(7.1), Inches(1.5), Inches(5.7), Inches(2.8), bg=SURF, border=CYAN)
    txbox(sl, "WHAT YOU WILL SEE",
          Inches(7.28), Inches(1.58), Inches(5.2), Inches(0.2),
          font_size=Pt(7.5), font_color=CYAN, bold=True)
    see_items = [
        "✦  Alert queue & webhook ingestion",
        "✦  Case creation + playbook tasks",
        "✦  Document upload → AI risk analysis",
        "✦  Transaction flow & flagged entries",
        "✦  Findings with 4-eye enforcement",
        "✦  AI narrative generation (live)",
        "✦  STR filing + goAML XML export",
        "✦  AI audit log (every model decision)",
        "✦  Hash-chain timeline",
    ]
    y = 1.82
    for item in see_items:
        txbox(sl, item, Inches(7.28), Inches(y), Inches(5.2), Inches(0.24),
              font_size=Pt(9.5), font_color=SUBTLE)
        y += 0.24

    # Running on card
    box(sl, Inches(7.1), Inches(4.45), Inches(5.7), Inches(1.1), bg=SURF, border=BORDER)
    txbox(sl, "RUNNING ON",
          Inches(7.28), Inches(4.53), Inches(5.2), Inches(0.18),
          font_size=Pt(7.5), font_color=MUTED, bold=True)
    txbox(sl, "Go API · Next.js · PostgreSQL · Python AI service\nllama3.2:3b — on-premise, air-gapped. No external API calls. Zero data egress.",
          Inches(7.28), Inches(4.73), Inches(5.2), Inches(0.7),
          font_size=Pt(9.5), font_color=SUBTLE)


# ══════════════════════════════════════════════════════════════════════════════
# SLIDE 6 — ARCHITECTURE
# ══════════════════════════════════════════════════════════════════════════════
def slide6(prs):
    sl = blank_slide(prs)
    fill_bg(sl)
    header(sl, "06 / 08")

    txbox(sl, "Enterprise Architecture",
          Inches(0.5), Inches(0.65), Inches(10), Inches(0.42),
          font_size=Pt(26), font_color=WHITE, bold=True)
    txbox(sl, "On-premise. Air-gapped AI. API-first. Deployed in one day.",
          Inches(0.5), Inches(1.1), Inches(10), Inches(0.28),
          font_size=Pt(11), font_color=SUBTLE)

    arch_text = (
        "INBOUND\n"
        "  TMS Alert  ── HMAC Webhook──→  /alerts/ingest\n"
        "  Decentro   ── HMAC Webhook──→  /inbound/events\n"
        "  Bulk CSV   ── Upload ────────→  /imports\n"
        "\n"
        "GO API  ·  ~150 endpoints  ·  Port 8080\n"
        "  Cases · Findings · STR · Narrative · Subjects · Playbooks\n"
        "  SLA · Analytics · Webhooks · RBAC · MFA · API Keys · Rate Limits\n"
        "\n"
        "POSTGRESQL          PYTHON AI SERVICE          MINIO / S3\n"
        "54 tables           /ask  /agent               Document storage\n"
        "pgvector            /narrative/generate         AES-256 SSE\n"
        "hash-chain          /process-doc\n"
        "AES-256 PII                  ↓\n"
        "                    OLLAMA — llama3.2:3b\n"
        "                    On-premise. Zero data egress.\n"
        "\n"
        "OUTBOUND\n"
        "  Webhooks on case events  ·  goAML 4.0 XML  ·  Scheduled reports"
    )
    box(sl, Inches(0.5), Inches(1.5), Inches(7.8), Inches(4.8),
        bg=RGBColor(0x02, 0x0b, 0x14), border=RGBColor(0x0f, 0x2a, 0x40))
    txbox(sl, arch_text, Inches(0.65), Inches(1.6), Inches(7.5), Inches(4.6),
          font_size=Pt(8.5), font_color=RGBColor(0x7d, 0xd3, 0xfc))

    # Security layers
    sec = [
        ("🔐  MFA (TOTP + backup codes)", "TOTP 2FA, JWT refresh tokens, scoped API keys, session limits at DB level"),
        ("🔒  AES-256 Column Encryption", "PII encrypted at rest in PostgreSQL. Key in environment, not in schema."),
        ("⛓️  Hash-Chain Audit Log", "prev_hash chaining on every audit record. Tamper breaks the chain detectably."),
        ("👥  RBAC — 5 Roles", "Viewer · Analyst · Manager · Admin · Super Admin. Enforced per endpoint."),
        ("🤖  AI Act Audit Trail", "Every AI decision: model version, full prompt, output, human review status, reviewer ID."),
        ("🚦  Account Lockout + Rate Limits", "Auto-lockout on failed logins. Per-IP rate limits on all auth endpoints."),
    ]
    y = 1.5
    for title, body in sec:
        box(sl, Inches(8.5), Inches(y), Inches(4.5), Inches(0.7), bg=SURF, border=BORDER)
        txbox(sl, title, Inches(8.65), Inches(y + 0.06), Inches(4.1), Inches(0.24),
              font_size=Pt(9.5), font_color=WHITE, bold=True)
        txbox(sl, body, Inches(8.65), Inches(y + 0.3), Inches(4.1), Inches(0.34),
              font_size=Pt(8.5), font_color=SUBTLE)
        y += 0.78


# ══════════════════════════════════════════════════════════════════════════════
# SLIDE 7 — BUSINESS OUTCOMES
# ══════════════════════════════════════════════════════════════════════════════
def slide7(prs):
    sl = blank_slide(prs)
    fill_bg(sl)
    header(sl, "07 / 08")

    txbox(sl, "Outcomes, Not Features",
          Inches(0.5), Inches(0.65), Inches(10), Inches(0.42),
          font_size=Pt(26), font_color=WHITE, bold=True)
    txbox(sl, "What your compliance leadership measures. What changes from day one.",
          Inches(0.5), Inches(1.1), Inches(10), Inches(0.28),
          font_size=Pt(11), font_color=SUBTLE)

    # Table header
    cols = ["What Matters", "Before OpsOwl", "With OpsOwl", "Driven by"]
    col_widths = [Inches(2.8), Inches(3.2), Inches(3.5), Inches(2.6)]
    x = Inches(0.4)
    y = 1.5
    for i, (col, w) in enumerate(zip(cols, col_widths)):
        box(sl, x, Inches(y), w, Inches(0.3), bg=SURF2, border=BORDER)
        txbox(sl, col, x + Inches(0.1), Inches(y + 0.05), w - Inches(0.2), Inches(0.22),
              font_size=Pt(8), font_color=MUTED, bold=True)
        x += w

    rows = [
        ("Investigation cycle time", "5–8 days average", "Target under 48 hours", "SLA enforcement"),
        ("STR narrative writing", "3–4 hours per STR", "AI draft in 2 min · Review 15 min", "AI narrative"),
        ("RBI examination readiness", "2–3 weeks of documents", "One-click hash-chain export", "Audit trail"),
        ("Analyst process standardization", "Every analyst differs", "Playbook enforces same process", "Playbooks"),
        ("Evidence linkage to STR", "Buried in email threads", "Document → Finding → STR", "Evidence gate"),
        ("Maker-checker compliance", "Email approvals, no record", "Enforced in code on every step", "4-eye rule"),
        ("Analyst investigation capacity", "8–10 active cases", "20+ with AI assistance", "AI Copilot"),
    ]
    y = 1.8
    for row in rows:
        x = Inches(0.4)
        for i, (cell, w) in enumerate(zip(row, col_widths)):
            bdr_col = BORDER if y % 0.4 < 0.2 else RGBColor(0x0a, 0x18, 0x27)
            box(sl, x, Inches(y), w, Inches(0.5), bg=SURF, border=BORDER)
            cell_col = SUBTLE
            if i == 1:
                cell_col = RED
            elif i == 2:
                cell_col = GREEN
            elif i == 0:
                cell_col = WHITE
            txbox(sl, cell, x + Inches(0.08), Inches(y + 0.06), w - Inches(0.16), Inches(0.38),
                  font_size=Pt(9), font_color=cell_col, bold=(i==0))
            x += w
        y += 0.5

    # Quote
    box(sl, Inches(0.4), Inches(5.4), Inches(12.5), Inches(0.7), bg=SURF, border=CYAN)
    txbox(sl,
          '"The question is not whether you can afford OpsOwl." The question is: can you afford the next RBI examination '
          "with your investigation trail living in a shared drive full of Word documents and email threads?",
          Inches(0.6), Inches(5.5), Inches(12.0), Inches(0.55),
          font_size=Pt(10.5), font_color=SUBTLE, italic=True)


# ══════════════════════════════════════════════════════════════════════════════
# SLIDE 8 — WHY OPSOWL + CTA
# ══════════════════════════════════════════════════════════════════════════════
def slide8(prs):
    sl = blank_slide(prs)
    fill_bg(sl)
    header(sl, "08 / 08")

    txbox(sl, "Why OpsOwl",
          Inches(0.5), Inches(0.65), Inches(10), Inches(0.42),
          font_size=Pt(28), font_color=WHITE, bold=True)
    txbox(sl, "Four differentiators. One next step.",
          Inches(0.5), Inches(1.1), Inches(10), Inches(0.28),
          font_size=Pt(11), font_color=SUBTLE)

    diff_cards = [
        ("🇮🇳", "India-Native. Regulation-Ready.", CYAN,
         "FIU-IND STR format, FINnet XML, PMLA Section 12 language in every AI-generated narrative. "
         "goAML 4.0 for multi-jurisdiction deployments. Not a US product retrofitted for India. Built here, for RBI."),
        ("🔒", "Air-Gapped AI. Zero Data Egress.", GREEN,
         "KYC documents, transaction data, subject PII — nothing leaves your server. LLM runs on-premise via Ollama. "
         "Compliance team AND IT security can both say yes. Switch to cloud AI with one env var when needed."),
        ("👥", "Human-in-the-Loop. By Architecture.", AMBER,
         "AI assists at every step. Humans approve at every step. Maker-checker is enforced in code — not policy. "
         "Creator cannot approve their own work. Every AI decision logged with model version and reviewer identity."),
        ("⛓️", "One System of Record. Audit-First.", BORDER,
         "Case → Evidence → Finding → Narrative → STR → Closure. Hash-chain tamper evidence on every record. "
         "When RBI walks in, you open OpsOwl — not a shared drive. Per-case report: one click, examination-ready."),
    ]
    card_w = Inches(3.1)
    x = Inches(0.4)
    for icon, title, col, body in diff_cards:
        box(sl, x, Inches(1.5), card_w, Inches(2.5), bg=SURF, border=col)
        txbox(sl, icon, x + Inches(0.15), Inches(1.58), card_w - Inches(0.3), Inches(0.35),
              font_size=Pt(22))
        txbox(sl, title, x + Inches(0.15), Inches(1.95), card_w - Inches(0.3), Inches(0.35),
              font_size=Pt(11), font_color=WHITE, bold=True)
        txbox(sl, body, x + Inches(0.15), Inches(2.33), card_w - Inches(0.3), Inches(1.55),
              font_size=Pt(9), font_color=SUBTLE)
        x += card_w + Inches(0.17)

    # CTA card
    box(sl, Inches(0.4), Inches(4.15), Inches(12.5), Inches(1.5), bg=SURF, border=CYAN)
    txbox(sl, "THE ASK",
          Inches(0.6), Inches(4.23), Inches(8), Inches(0.2),
          font_size=Pt(7.5), font_color=CYAN, bold=True)
    txbox(sl, "Start a 30-Day Pilot",
          Inches(0.6), Inches(4.45), Inches(8), Inches(0.3),
          font_size=Pt(15), font_color=WHITE, bold=True)
    txbox(sl,
          "Share one sample alert payload from your TMS. We configure the field mapping. "
          "Webhook goes live. Your compliance team runs 10–20 real investigations in OpsOwl. "
          "You decide if it belongs in your stack.",
          Inches(0.6), Inches(4.78), Inches(7.8), Inches(0.7),
          font_size=Pt(10.5), font_color=SUBTLE)

    # Stats
    box(sl, Inches(9.0), Inches(4.2), Inches(1.7), Inches(1.35), bg=SURF2, border=BORDER)
    txbox(sl, "48h", Inches(9.0), Inches(4.28), Inches(1.7), Inches(0.55),
          font_size=Pt(30), font_color=CYAN, bold=True, align=PP_ALIGN.CENTER)
    txbox(sl, "to your first live case", Inches(9.0), Inches(4.82), Inches(1.7), Inches(0.55),
          font_size=Pt(8.5), font_color=MUTED, align=PP_ALIGN.CENTER)

    box(sl, Inches(10.8), Inches(4.2), Inches(1.7), Inches(1.35), bg=SURF2, border=BORDER)
    txbox(sl, "1", Inches(10.8), Inches(4.28), Inches(1.7), Inches(0.55),
          font_size=Pt(30), font_color=CYAN, bold=True, align=PP_ALIGN.CENTER)
    txbox(sl, "day to deploy on-premise", Inches(10.8), Inches(4.82), Inches(1.7), Inches(0.55),
          font_size=Pt(8.5), font_color=MUTED, align=PP_ALIGN.CENTER)


# ── BUILD ─────────────────────────────────────────────────────────────────────
prs = new_prs()
slide1(prs)
slide2(prs)
slide3(prs)
slide4(prs)
slide5(prs)
slide6(prs)
slide7(prs)
slide8(prs)

out = "/Users/nagurureddybasha/Documents/investigation-workspace/docs/decentro-demo/OpsOwl-Decentro-Pitch.pptx"
prs.save(out)
print(f"Saved: {out}")
