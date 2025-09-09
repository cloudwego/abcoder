import * as _ from 'lodash';
import * as fs from 'fs';
import * as path from 'path';
import * as JSON5 from 'json5';

const MagicNumber = 1e9 + 7;

type ABC<T> = ABC<T>[]

export class Calculator {
  private history: Array<{ operation: string; result: number; timestamp: Date }> = [];

  add(a: number, b: number): number {
    const result = a + b;
    this.logOperation('add', result, [a, b]);
    return result;
  }

  addMagicNumber(a: number): number {
    const result = a + MagicNumber;
    this.logOperation('addMagicNumber', result, [a]);
    return result;
  }

  subtract(a: number, b: number): number {
    const result = a - b;
    this.logOperation('subtract', result, [a, b]);
    return result;
  }

  multiply(a: number, b: number): number {
    const result = a * b;
    this.logOperation('multiply', result, [a, b]);
    return result;
  }

  divide(a: number, b: number): number {
    if (b === 0) {
      throw new Error('Division by zero is not allowed');
    }
    const result = a / b;
    this.logOperation('divide', result, [a, b]);
    return result;
  }

  power(base: number, exponent: number): number {
    const result = Math.pow(base, exponent);
    this.logOperation('power', result, [base, exponent]);
    return result;
  }

  factorial(n: number): number {
    if (n < 0) {
      throw new Error('Factorial is not defined for negative numbers');
    }
    if (n === 0 || n === 1) {
      return 1;
    }
    
    let result = 1;
    for (let i = 2; i <= n; i++) {
      result *= i;
    }
    
    this.logOperation('factorial', result, [n]);
    return result;
  }

  sum(numbers: number[]): number {
    const result = _.sum(numbers);
    this.logOperation('sum', result, numbers);
    return result;
  }

  average(numbers: number[]): number {
    if (numbers.length === 0) {
      throw new Error('Cannot calculate average of empty array');
    }
    const result = _.sum(numbers) / numbers.length;
    this.logOperation('average', result, numbers);
    return result;
  }

  median(numbers: number[]): number {
    const sorted = _.sortBy(numbers);
    const len = sorted.length;
    
    if (len === 0) {
      throw new Error('Cannot calculate median of empty array');
    }
    
    let result: number;
    if (len % 2 === 0) {
      result = (sorted[len / 2 - 1] + sorted[len / 2]) / 2;
    } else {
      result = sorted[Math.floor(len / 2)];
    }
    
    this.logOperation('median', result, numbers);
    return result;
  }

  private logOperation(operation: string, result: number, inputs: number[]): void {
    this.history.push({
      operation,
      result,
      timestamp: new Date()
    });

    // Limit history to last 100 operations
    if (this.history.length > 100) {
      this.history = this.history.slice(-100);
    }
  }

  getHistory(): Array<{ operation: string; result: number; timestamp: Date }> {
    return [...this.history];
  }

  clearHistory(): void {
    this.history = [];
  }

  saveHistoryToFile(filename: string): void {
    const filePath = path.join(__dirname, '../../logs', filename);
    const data = JSON.stringify(this.history, null, 2);
    
    try {
      if (!fs.existsSync(path.dirname(filePath))) {
        fs.mkdirSync(path.dirname(filePath), { recursive: true });
      }
      fs.writeFileSync(filePath, data);
    } catch (error) {
      console.error('Error saving history:', error);
    }
  }

  loadHistoryFromFile(filename: string): void {
    const filePath = path.join(__dirname, '../../logs', filename);
    
    try {
      if (fs.existsSync(filePath)) {
        const data = fs.readFileSync(filePath, 'utf-8');
        this.history = JSON5.parse(data);
      }
    } catch (error) {
      console.error('Error loading history:', error);
    }
  }
}