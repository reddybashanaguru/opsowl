// Emirates ID: 784-YYYY-XXXXXXX-X  (15 digits; check digit uses Luhn algorithm)
// Country code 784 is UAE in ISO 3166-1 numeric.

const EID_REGEX = /^784-\d{4}-\d{7}-\d$/;

/**
 * Returns true when the Emirates ID passes format and Luhn check.
 */
export function validateEmiratesID(raw: string): boolean {
  if (!EID_REGEX.test(raw)) return false;
  const digits = raw.replace(/-/g, "");
  return luhn(digits);
}

/**
 * Formats a raw 15-digit string into 784-YYYY-XXXXXXX-X display form.
 * Accepts digits with or without existing dashes.
 */
export function formatEmiratesID(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 15);
  if (digits.length <= 3) return digits;
  if (digits.length <= 7) return `${digits.slice(0, 3)}-${digits.slice(3)}`;
  if (digits.length <= 14) return `${digits.slice(0, 3)}-${digits.slice(3, 7)}-${digits.slice(7)}`;
  return `${digits.slice(0, 3)}-${digits.slice(3, 7)}-${digits.slice(7, 14)}-${digits.slice(14)}`;
}

/**
 * Error message for UI display, or null if valid.
 */
export function emiratesIDError(raw: string): string | null {
  if (!raw) return null;
  const digits = raw.replace(/\D/g, "");
  if (digits.length < 15) return "Emirates ID must be 15 digits (784-YYYY-XXXXXXX-X)";
  if (!EID_REGEX.test(raw)) return "Format must be 784-YYYY-XXXXXXX-X";
  if (!luhn(digits)) return "Emirates ID check digit is invalid";
  return null;
}

// Standard Luhn algorithm (mod-10)
function luhn(digits: string): boolean {
  let sum = 0;
  let double = false;
  for (let i = digits.length - 1; i >= 0; i--) {
    let d = parseInt(digits[i], 10);
    if (double) {
      d *= 2;
      if (d > 9) d -= 9;
    }
    sum += d;
    double = !double;
  }
  return sum % 10 === 0;
}
