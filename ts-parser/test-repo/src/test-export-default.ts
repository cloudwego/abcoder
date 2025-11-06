// 测试用例:箭头函数先声明,然后作为默认导出
const foo = () => {
  console.log('bar')
}

export default foo;

// 对比:直接导出的箭头函数
export const bar = () => {
  console.log('baz')
}
