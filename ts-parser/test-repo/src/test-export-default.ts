// 测试用例:箭头函数先声明,然后作为默认导出
const foo = () => {
  console.log('bar')
}

export default foo;

// 对比:直接导出的箭头函数
export const bar = () => {
  console.log('baz')
}

export type Status = 'normal' | 'abnormal'

export type Result<T> = T | Status

export type ServerStatus = {
  code: number;
  status: Status;
}

export const convert = <T>(s: T): Result<T> => {
  // 如果输入是字符串，返回 'normal'，否则返回输入本身
  if (typeof s === 'string') {
    return 'normal';
  }
  return s;
};

export const flipStatus = (s: Status): Result<Status> => {
  return s === 'normal' ? 'abnormal' : 'normal';
}
