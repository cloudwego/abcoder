package entity

type MyStructC struct {
}

type MyStructD struct {
}

type MyStruct struct {
	a string
	b string
	c MyStructC
	MyStructD
}

func (a MyStruct) String() string {
	return "base struct"
}

func (c MyStructC) String() string {
	return "I'm struct c"
}

func (c MyStructD) String() string {
	return "I'm struct d"
}

func A() {
	return
}
