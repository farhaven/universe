package orrery

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"../vector"
)

type Planet struct {
	T   uint64
	R   float64
	M   float64
	Pos vector.V3
	Vel vector.V3

	Trail []vector.V3

	L sync.Mutex
}

type Orrery struct {
	planets     []*Planet
	trailLength int
	q           chan bool
	l           sync.Mutex
	looptime    time.Duration
}

func (o *Orrery) Planets() []*Planet {
	o.l.Lock()
	defer o.l.Unlock()

	r := make([]*Planet, len(o.planets))
	copy(r, o.planets)

	return r
}

func (p Planet) String() string {
	return fmt.Sprintf(`R:%.2f, M:%.2f, Pos:%s, Vel:%s`, p.R, p.M, p.Pos, p.Vel)
}

func (p *Planet) move(trailLength int) {
	p.L.Lock()
	defer p.L.Unlock()

	newPos := p.Pos.Add(p.Vel)

	addToTrail := false

	if len(p.Trail) > 0 {
		d := newPos.Distance(p.Trail[len(p.Trail)-1])

		if d > p.R {
			addToTrail = true
		}
	} else {
		addToTrail = true
	}

	if addToTrail {
		p.Trail = append(p.Trail, p.Pos)
		if len(p.Trail) > trailLength {
			p.Trail = p.Trail[len(p.Trail)-trailLength:]
		}
	}

	p.Pos = newPos
}

func (p *Planet) applyForce(f vector.V3, s float64) {
	p.Vel = p.Vel.Add(f.Scaled(s / p.M))
}

type planetIntersection int

const (
	TOTAL planetIntersection = iota
	PARTIAL
	NONE
)

func (i planetIntersection) String() string {
	switch i {
	case TOTAL:
		return "TOTAL"
	case PARTIAL:
		return "PARTIAL"
	case NONE:
		return "NONE"
	default:
		return "UNKNOWN"
	}
}

func (p *Planet) collide(px *Planet) planetIntersection {
	p.L.Lock()
	defer p.L.Unlock()

	px.L.Lock()
	defer px.L.Unlock()

	d := p.Pos.Distance(px.Pos)
	if d > p.R+px.R {
		return NONE
	}

	if d < math.Min(p.R, px.R) {
		return TOTAL
	}

	CR := 0.5 // Coefficient of restitution, 0: totally elastic, 1: totally inelastic

	v1 := p.Vel.Length()
	v2 := px.Vel.Length()
	if math.IsNaN(v1) || math.IsNaN(v2) || math.IsInf(v1, 0) || math.IsInf(v2, 0) {
		panic(fmt.Sprintf(`v1: (%v %v) v2: (%v %v)`, p.Vel, v1, px.Vel, v2))
	}

	a1 := (CR*px.M*(v2-v1) + p.M*v1 + px.M*v2) / (p.M + px.M)
	d1 := p.Pos.Sub(px.Pos)
	if d1.Length() == 0 {
		panic(`zero distance`)
	}
	x1 := p.Vel.Sub(px.Vel).Dot(d1) / d1.Length()
	n1 := a1 * x1
	if math.IsNaN(n1) {
		txt := fmt.Sprintf(`%v|%v|%v|%v`, CR, px.M*(v2-v1), p.M*v1, px.M*v2)
		panic(fmt.Sprintf(`%s, a1: %v x1: %v`, txt, a1, x1))
	}
	p.applyForce(d1.Scaled(n1), 1)

	a2 := (CR*p.M*(v1-v2) + p.M*v1 + px.M*v2) / (p.M + px.M)
	d2 := px.Pos.Sub(p.Pos)
	if d2.Length() == 0 {
		panic(`zero distance`)
	}
	x2 := p.Vel.Sub(px.Vel).Dot(d2) / d2.Length()
	n2 := a2 * x2
	if math.IsNaN(n2) {
		txt := fmt.Sprintf(`%v|%v|%v|%v`, CR, px.M*(v1-v2), p.M*v1, px.M*v2)
		panic(fmt.Sprintf(`%s, a1: %v x1: %v`, txt, a2, x2))
	}
	p.applyForce(d2.Scaled(n2), 1)

/*
	if d < math.Max(p.R, px.R) {
		return TOTAL
	}
*/

	return PARTIAL
}

func (p *Planet) interactGravity(px *Planet) {
	p.L.Lock()
	defer p.L.Unlock()

	px.L.Lock()
	defer px.L.Unlock()

	if p.M == 0 || px.M == 0 {
		return
	}

	if px == p {
		panic(`can't gravitationally interact with myself!`)
	}

	// G := 6.67 * math.Pow(10, -11)
	G := float64(0.05)

	v := px.Pos.Sub(p.Pos)

	d := math.Max(1, v.Length())

	M := p.M + px.M
	a := (G * M) / (d * d)

	v = v.Normalized().Scaled(a)

	p.applyForce(v, 1)
	px.applyForce(v, -1)
}

func (o *Orrery) loop() {
	for {
		t_start := time.Now()
		o.l.Lock()

		pchan := make(chan [2]*Planet)
		wg := sync.WaitGroup{}
		gw := func() {
			for p := range pchan {
				p[0].interactGravity(p[1])
				wg.Done()
			}
		}
		for i := 0; i < 4; i++ {
			go gw()
		}
		for i, p := range o.planets {
			for _, px := range o.planets[i+1:] {
				wg.Add(1)
				pchan <- [2]*Planet{p, px}
			}
		}
		wg.Wait()

		for _, p := range o.planets {
			p.move(o.trailLength)
		}

		// Check for collisions
		for i, p := range o.planets[:len(o.planets)-1] {
			for _, px := range o.planets[i+1:] {
				if c := p.collide(px); c == TOTAL {
					l, s := p, px
					if l.M < s.M {
						l, s = px, p
					}
					s.Pos = l.Pos
					s.Vel = s.Vel
				}
			}
		}

		o.l.Unlock()

		t_sleep := o.looptime.Nanoseconds() - time.Since(t_start).Nanoseconds()
		if t_sleep > 0 {
			time.Sleep(time.Duration(t_sleep) * time.Nanosecond)
		}
	}
}

func (o *Orrery) SpawnPlanet(p vector.V3) {
	o.l.Lock()
	defer o.l.Unlock()

	o.planets = append(o.planets, &Planet{T: 0, R: 1.0, M: 5, Pos: p})
}

func (o *Orrery) SpawnVolume(p vector.V3) {
	o.l.Lock()
	defer o.l.Unlock()

	rn := func(r float64) float64 {
		return (rand.Float64() - 0.5) * r
	}

	for i := 0; i < 10; i++ {
		px := vector.V3{p.X + rn(100), p.Y + rn(100), p.Z + rn(100)}
		o.planets = append(o.planets, &Planet{T: 0, R: 1.0, M: 2, Pos: px})
	}
}

func New() *Orrery {
	o := &Orrery{
		planets: []*Planet{
			&Planet{T: 0, R: 30.0, M: 500.972},                                             // Earth
			&Planet{T: 0, R: 5, M: 7.3459, Pos: vector.V3{X: 200}, Vel: vector.V3{Y: 0.1}}, // Moon
		},
		trailLength: 20,
		looptime:    5 * time.Millisecond,

		q: make(chan bool),
	}

	go o.loop()

	return o
}
