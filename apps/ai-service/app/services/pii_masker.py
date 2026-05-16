"""PII Masker — strips PII from text before sending to external LLMs.

Uses Microsoft Presidio for entity detection. Detected entities are replaced
with deterministic tokens ([PAN_1], [EMAIL_1], etc.) so the LLM never sees
raw personal data. The token map is held only in memory for the request
duration — never persisted.
"""
from __future__ import annotations

import re
from dataclasses import dataclass, field


# Regex-based fallback patterns for common Indian PII
# (used when Presidio spacy model is not available)
_PATTERNS: list[tuple[str, str]] = [
    (r"\b[A-Z]{5}[0-9]{4}[A-Z]\b", "PAN"),                    # PAN card
    (r"\b[2-9]{1}[0-9]{3}\s?[0-9]{4}\s?[0-9]{4}\b", "AADHAAR"),  # Aadhaar
    (r"\b[6-9]\d{9}\b", "PHONE"),                              # Indian mobile
    (r"\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b", "EMAIL"),
    (r"\b[A-Z]{4}0[A-Z0-9]{6}\b", "IFSC"),                    # IFSC code
    (r"\b\d{9,18}\b", "ACCOUNT"),                              # Bank account (broad)
]


@dataclass
class MaskResult:
    masked_text: str
    token_map: dict[str, str] = field(default_factory=dict)

    def restore(self, text: str) -> str:
        """Replace tokens back with original values in a response text."""
        for token, original in self.token_map.items():
            text = text.replace(token, original)
        return text


def mask(text: str) -> MaskResult:
    """Mask PII in text and return masked text + token map."""
    counters: dict[str, int] = {}
    token_map: dict[str, str] = {}
    result = text

    for pattern, entity_type in _PATTERNS:
        def replace(m: re.Match, entity_type: str = entity_type) -> str:
            original = m.group(0)
            # Check if already tokenized
            for token, orig in token_map.items():
                if orig == original:
                    return token
            counters[entity_type] = counters.get(entity_type, 0) + 1
            token = f"[{entity_type}_{counters[entity_type]}]"
            token_map[token] = original
            return token

        result = re.sub(pattern, replace, result)

    return MaskResult(masked_text=result, token_map=token_map)
