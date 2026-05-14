package org.example;

public class Dog implements Animal {
    private final String n;

    public Dog(String n) {
        this.n = n;
    }

    @Override
    public void eat() {
        System.out.println(n + " eats.");
    }

    @Override
    public String name() {
        return n;
    }
}
