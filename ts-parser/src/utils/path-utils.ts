import * as path from 'path';

export class PathUtils {
  private projectRoot: string;

  constructor(projectRoot: string) {
    this.projectRoot = projectRoot;
  }

  /**
   * Get relative path from project root
   */
  getRelativePath(filePath: string): string {
    return path.relative(this.projectRoot, filePath).replace(/\\/g, '/');
  }

  /**
   * Get package path relative to project root
   */
  getPkgPath(packagePath: string): string {
    // Resolve the provided path against the project root to get a full, absolute path.
    const absolutePath = path.resolve(this.projectRoot, packagePath);

    // If the resolved path is the same as the project root, it's the base package.
    if (absolutePath === this.projectRoot) {
      return '.';
    }

    // Otherwise, compute the relative path from the project root.
    const relativePath = path.relative(this.projectRoot, absolutePath);

    // Normalize slashes for consistency.
    return relativePath.replace(/\\/g, '/');
  }
}