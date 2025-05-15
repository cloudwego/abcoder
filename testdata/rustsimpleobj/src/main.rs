struct Foo(u32);

impl Foo {
    pub fn new(value: u32) -> Self {
        Foo(value)
    }

    pub fn bar(&mut self, increment: u32) {
        self.0 += increment;
    }

    pub fn faz(&mut self, decrement: u32) {
        self.0 -= decrement;
    }
}

fn main() {
    let mut my_foo = Foo::new(10);
    my_foo.bar(5);
    my_foo.faz(5);
}
