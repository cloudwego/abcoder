package org.example;

public class Dog implements Animal {

    public String field;

    public void fetch() {
        System.out.println("Fetching the ball!");
    }

    @Override
    public String makeSound() {
        return "Woof!";
    }
}