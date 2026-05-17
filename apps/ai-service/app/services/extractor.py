"""Text extraction from documents. Supports PDF, DOCX, XLSX, TXT, PNG/JPG."""
from __future__ import annotations

import io
from dataclasses import dataclass, field


@dataclass
class ExtractionResult:
    text: str
    page_count: int | None = None
    error: str | None = None


def extract(content: bytes, file_type: str, mime_type: str) -> ExtractionResult:
    """Dispatch extraction based on file_type. Returns raw text."""
    ft = file_type.lower()
    try:
        if ft == "pdf":
            return _extract_pdf(content)
        elif ft in ("docx", "doc"):
            return _extract_docx(content)
        elif ft in ("xlsx", "xls"):
            return _extract_xlsx(content)
        elif ft == "txt":
            return ExtractionResult(text=content.decode("utf-8", errors="replace"))
        elif ft in ("png", "jpg", "jpeg"):
            return _extract_image(content)
        else:
            return ExtractionResult(text="", error=f"unsupported file type: {ft}")
    except Exception as exc:
        return ExtractionResult(text="", error=str(exc))


def _extract_pdf(content: bytes) -> ExtractionResult:
    import fitz  # PyMuPDF

    doc = fitz.open(stream=content, filetype="pdf")
    pages = []
    for page in doc:
        text = page.get_text("text")
        if text.strip():
            pages.append(text)
        else:
            # Fallback: render page as image and OCR
            try:
                pix = page.get_pixmap(dpi=150)
                img_bytes = pix.tobytes("png")
                ocr_result = _ocr_image(img_bytes)
                if ocr_result:
                    pages.append(ocr_result)
            except Exception:
                pass
    return ExtractionResult(text="\n\n".join(pages), page_count=len(doc))


def _extract_docx(content: bytes) -> ExtractionResult:
    from docx import Document

    doc = Document(io.BytesIO(content))
    paragraphs = [p.text for p in doc.paragraphs if p.text.strip()]
    return ExtractionResult(text="\n\n".join(paragraphs))


def _extract_xlsx(content: bytes) -> ExtractionResult:
    import openpyxl

    wb = openpyxl.load_workbook(io.BytesIO(content), read_only=True, data_only=True)
    parts: list[str] = []
    for sheet in wb.worksheets:
        rows: list[str] = []
        for row in sheet.iter_rows(values_only=True):
            cells = [str(c) if c is not None else "" for c in row]
            if any(c.strip() for c in cells):
                rows.append("\t".join(cells))
        if rows:
            parts.append(f"[Sheet: {sheet.title}]\n" + "\n".join(rows))
    return ExtractionResult(text="\n\n".join(parts))


def _extract_image(content: bytes) -> ExtractionResult:
    text = _ocr_image(content) or ""
    return ExtractionResult(text=text)


def _ocr_image(content: bytes) -> str:
    import pytesseract
    from PIL import Image

    img = Image.open(io.BytesIO(content))
    return pytesseract.image_to_string(img)
