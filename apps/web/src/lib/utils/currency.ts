// Currency formatting utilities — jurisdiction-aware.
// AED uses "en-AE" locale; INR uses "en-IN"; fallback to "en-US".

const LOCALE_MAP: Record<string, string> = {
  AED: "en-AE",
  INR: "en-IN",
  USD: "en-US",
  EUR: "en-DE",
  GBP: "en-GB",
  KES: "en-KE",
  SAR: "ar-SA",
  QAR: "ar-QA",
};

function localeFor(currency: string): string {
  return LOCALE_MAP[currency.toUpperCase()] ?? "en-US";
}

/**
 * Full currency format: AED 55,000.00 / ₹ 10,00,000.00
 */
export function formatCurrency(amount: number, currency: string): string {
  if (!currency) return amount.toLocaleString();
  try {
    return new Intl.NumberFormat(localeFor(currency), {
      style: "currency",
      currency,
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(amount);
  } catch {
    return `${currency} ${amount.toLocaleString()}`;
  }
}

/**
 * Compact format: AED 55K / ₹ 10L / $1.2M
 */
export function formatCompact(amount: number, currency: string): string {
  if (!currency) return compactNumber(amount);
  try {
    return new Intl.NumberFormat(localeFor(currency), {
      style: "currency",
      currency,
      notation: "compact",
      maximumFractionDigits: 1,
    }).format(amount);
  } catch {
    return `${currency} ${compactNumber(amount)}`;
  }
}

function compactNumber(n: number): string {
  if (Math.abs(n) >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (Math.abs(n) >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return n.toLocaleString();
}

export type CTRStatus = "above" | "near" | "normal";

/**
 * Returns the CTR reporting status for a given amount.
 * "above"  — ≥ threshold (mandatory reporting)
 * "near"   — ≥ 90% of threshold (structuring risk)
 * "normal" — below 90% or threshold is 0 (disabled)
 */
export function getCTRStatus(amount: number, threshold: number): CTRStatus {
  if (threshold <= 0) return "normal";
  if (amount >= threshold) return "above";
  if (amount >= threshold * 0.9) return "near";
  return "normal";
}
