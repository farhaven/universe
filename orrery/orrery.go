package orrery

import "math"

type Vector3 struct {
	X, Y, Z float64
}

func (v *Vector3) add (o Vector3) {
	v.X += o.X
	v.Y += o.Y
	v.Z += o.Z
}

func (v *Vector3) sub (o Vector3) {
	v.X -= o.X
	v.Y -= o.Y
	v.Z -= o.Z
}

func (v *Vector3) scale (n float64) {
	v.X *= n
	v.Y *= n
	v.Z *= n
}

func (v *Vector3) scaled (n float64) Vector3 {
	return Vector3{ v.X * n, v.Y * n, v.Z * n }
}

func (v *Vector3) length() float64 {
	return math.Sqrt(v.X * v.X + v.Y * v.Y + v.Z * v.Z)
}

type Planet struct {
	R   float64
	M   float64
	Pos Vector3
	Vel Vector3
}

type Orrery struct {
	Planets []*Planet
}

func (p *Planet) move() {
	p.Pos.add(p.Vel)
}

func (p *Planet) affectGravity(o *Orrery) {
	// G := 6.67 * math.Pow(10, -11)
	G := float64(0.05)
	for _, px := range o.Planets {
		if p == px {
			continue
		}

		v := Vector3{ px.Pos.X, px.Pos.Y, px.Pos.Z }
		v.sub(p.Pos)

		d := math.Max(1, v.length())

		M := p.M + px.M
		a := (G * M) / (d * d)

		v.scale(a/v.length())

		p.Vel.add(v.scaled(1/p.M))
	}
}

func (o *Orrery) StepPlanets() {
	for _, p := range o.Planets {
		p.affectGravity(o)
	}

	for _, p := range o.Planets {
		p.move()
	}
}

func (o *Orrery) SpawnPlanet(x, y, z float64) {
	o.Planets = append(o.Planets, &Planet{R: 1.0, Pos: Vector3{x, y, z}})
}

func New () *Orrery {
	o := &Orrery{ Planets: []*Planet{
		&Planet{R: 30.0, M: 500.972}, // Earth
		&Planet{R: 5, M: 7.3459, Pos: Vector3{X: 400}, Vel: Vector3{Y: 0.1}}, // Moon
	},}

	return o
}
