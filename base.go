package tensile

type Base struct {
	Shape Shape
	Name  string
}

func (base Base) Identity() (Shape, string) {
	return base.Shape, base.Name
}
