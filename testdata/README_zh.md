# 测例规范
每个测例是对应语言的一个单独项目(目录)，放在 `testdata/{lang}/index_{name}` 里。
其中 index 是 0 开始的数字，go test 会按照这个顺序测试。

go test 中有时只会测试 0_xx 的测例。
