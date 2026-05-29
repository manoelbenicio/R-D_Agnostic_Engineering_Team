/**
 * Redacts a credential key showing only prefixes and the final 4 characters.
 * E.g., maskKey('sk-abc123XYZ') -> 'sk-…23XYZ' (or 'sk-…XYZ' showing last 4)
 */
export function maskKey(key: string): string {
  if (!key) return '';
  if (key.length <= 4) {
    return '…' + key.slice(-2);
  }
  const lastFour = key.slice(-4);
  const dashIndex = key.indexOf('-');
  if (dashIndex !== -1 && dashIndex < key.length - 5) {
    const prefix = key.substring(0, dashIndex + 1);
    return `${prefix}…${lastFour}`;
  }
  return `…${lastFour}`;
}
export default maskKey;
