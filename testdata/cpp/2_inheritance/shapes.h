#ifndef SHAPES_H
#define SHAPES_H

#include <string>

namespace shapes {

// Base class with virtual interface.
class Shape {
public:
    virtual ~Shape() = default;
    virtual double area() const = 0;
    virtual std::string describe() const;
};

// Auxiliary mixin used to demonstrate multiple inheritance.
class Drawable {
public:
    virtual ~Drawable() = default;
    virtual void draw() const;
};

// Single inheritance: `Circle` extends `Shape`.
class Circle : public Shape {
public:
    explicit Circle(double r) : radius_(r) {}
    double area() const override;
    std::string describe() const override;

private:
    double radius_;
};

// Single inheritance with a different access keyword + virtual base method.
class Square : public Shape {
public:
    explicit Square(double s) : side_(s) {}
    double area() const override;

private:
    double side_;
};

// Multiple inheritance: `LabeledCircle` extends `Circle` and `Drawable`.
class LabeledCircle : public Circle, public Drawable {
public:
    LabeledCircle(double r, std::string label)
        : Circle(r), label_(std::move(label)) {}
    void draw() const override;
    const std::string& label() const { return label_; }

private:
    std::string label_;
};

// Template base — Container<int> is the actual base, the `int` should NOT be
// picked up by the parser as a base class (it's a template argument).
template <typename T>
class Container {
public:
    void push(const T& v) { data_ = v; }
    T get() const { return data_; }

private:
    T data_{};
};

class IntStore : public Container<int> {
public:
    void inc() { push(get() + 1); }
};

} // namespace shapes

#endif // SHAPES_H
