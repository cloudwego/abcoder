/**
 * Common type utility functions
 */

export class TypeUtils {
  private static readonly PRIMITIVE_TYPES = new Set([
    'string', 'number', 'boolean', 'void', 'any', 'unknown', 'never', 
    'null', 'undefined', 'object', 'symbol', 'bigint'
  ]);

  /**
   * Check if a type name is a primitive type
   */
  static isPrimitiveType(typeName: string): boolean {
    return this.PRIMITIVE_TYPES.has(typeName.toLowerCase());
  }

  /**
   * Extract base type name from complex type expressions
   */
  static extractBaseTypeName(typeName: string): string {
    return typeName.split('<')[0].split('&')[0].split('|')[0].trim();
  }
}