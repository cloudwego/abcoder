import { Calculator } from '../utils/calculator';
import { User } from '../models/user';

describe('Calculator', () => {
  it('should add numbers correctly', () => {
    const calc = new Calculator();
    expect(calc.add(2, 3)).toBe(5);
  });
});
