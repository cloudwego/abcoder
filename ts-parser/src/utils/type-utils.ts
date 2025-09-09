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
}