export const SCHEMA_VERSION = 1;

export function isCompatible(doc: { schema_version: number }): boolean {
  return doc.schema_version <= SCHEMA_VERSION;
}
