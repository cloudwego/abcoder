#include "shapes.h"

namespace shapes {

std::string Shape::describe() const { return "shape"; }

void Drawable::draw() const {}

double Circle::area() const { return 3.14159 * radius_ * radius_; }

std::string Circle::describe() const { return "circle"; }

double Square::area() const { return side_ * side_; }

void LabeledCircle::draw() const {}

} // namespace shapes
