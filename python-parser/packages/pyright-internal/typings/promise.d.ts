/*
 * promise.d.ts
 * Type declarations for Promise prototype extensions
 */

declare global {
    interface Promise<T> {
        ignoreErrors(): void;
    }
}

export {};
