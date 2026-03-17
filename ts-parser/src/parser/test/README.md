# Parser Unit Tests

This directory contains comprehensive unit tests for the TypeScript parser modules with high logic coverage.

## Test Coverage

### FunctionParser.test.ts
- **Function declarations**: Regular, exported, default export
- **Class methods**: Instance methods, static methods, constructors
- **Arrow functions**: Variable assignments, complex expressions
- **Interface methods**: Method signatures with parameters and return types
- **Function calls**: Direct calls, method calls, chained calls
- **Type references**: Custom types, interfaces, generics
- **Global variables**: Cross-module references, built-in filtering
- **Edge cases**: Anonymous functions, destructuring, generics

### VarParser.test.ts
- **Variable declarations**: const, let, var with different scopes
- **Class properties**: Public, private, protected, static, readonly
- **Enum members**: String enums, numeric enums, const enums
- **Destructuring**: Object, array, nested destructuring
- **Type extraction**: Custom types, primitives, complex types
- **Dependencies**: Initializer dependencies, cross-references
- **Edge cases**: Uninitialized variables, complex type annotations

### TypeParser.test.ts
- **Class declarations**: Simple, exported, abstract, generic classes
- **Interface declarations**: Simple, exported, generic interfaces
- **Type aliases**: Simple, complex, union, intersection, generic aliases
- **Enum declarations**: String, numeric, const enums
- **Inheritance**: Class inheritance, interface inheritance, implementation
- **Type dependencies**: Property types, method signatures, complex types
- **Edge cases**: Anonymous classes, nested types, function types

## Running Tests

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage

# Run tests with verbose output
npm run test:verbose
```

## Coverage Report

After running `npm run test:coverage`, coverage reports will be available in:
- `coverage/lcov-report/index.html` - HTML report
- `coverage/lcov.info` - LCOV format for CI tools

## Test Structure

- **test-utils.ts**: Common test utilities and helper functions
- **setup.ts**: Global test setup and configuration
- **jest.config.js**: Jest configuration for TypeScript testing
- **temp/**: Temporary directory for test files (auto-cleaned)

## Writing New Tests

1. Create test file with `.test.ts` extension
2. Use `createTestProject` utility for creating test projects
3. Follow the existing test patterns for consistency
4. Ensure high coverage for new functionality
5. Test both happy paths and edge cases