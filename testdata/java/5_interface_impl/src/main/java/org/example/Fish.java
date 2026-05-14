package org.example;

public class Fish implements Animal, Swimmer {
    private final String n;

    public Fish(String n) {
        this.n = n;
    }

    @Override
    public void eat() {
        System.out.println(n + " eats.");
    }

    @Override
    public void swim() {
        System.out.println(n + " swims.");
    }

    @Override
    public String name() {
        return n;
    }
}
