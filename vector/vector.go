package vector

import "math"

type V3 struct {
	X, Y, Z float64
}
func (v V3) Length() float64 {
	return math.Sqrt(v.X * v.X + v.Y * v.Y + v.Z * v.Z)
}
func (v V3) Dot(o V3) float64 {
	return v.X * o.X + v.Y * o.Y + v.Z * o.Z
}

func (v V3) Cross(o V3) V3 {
	return V3{
		v.Y * o.Z - o.Y * v.Z,
		o.X * v.Z - v.X * o.Z,
		v.X * o.Y - o.X * v.Y,
	}
}
func (v V3) Normalized() V3 {
	return v.Scaled(1/v.Length())
}
func (v V3) Sub(o V3) V3 {
	return V3 { v.X - o.X, v.Y - o.Y, v.Z - o.Z }
}
func (v V3) Add(o V3) V3 {
	return V3 { v.X + o.X, v.Y + o.Y, v.Z + o.Z }
}
func (v V3) Scaled(n float64) V3 {
	return V3{v.X * n, v.Y * n, v.Z * n}
}
func (v V3) Distance(o V3) float64 {
	return v.Sub(o).Length()
}

type Plane [2]V3 // Normal, Point on plane
func (p *Plane) Distance(px V3) float64 {
	n  := p[0]
	p0 := p[1]

	D := n.Scaled(-1).Dot(p0)

	return n.Dot(px) + D
}
