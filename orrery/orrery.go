package orrery

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	"git.c3pb.de/farhaven/universe/vector"
)

type Particle struct {
	T   float64
	R   float64
	M   float64
	Pos vector.V3
	Vel vector.V3

	Trail []vector.V3

	L sync.Mutex
}

type command interface{}
type CommandSpawnParticle struct {
	Pos vector.V3
	M float64
}
type CommandSpawnVolume struct {
	Pos vector.V3
}
type CommandPause struct{}
type CommandLoad struct{}
type CommandStore struct{}
type Orrery struct {
	particles   []*Particle
	trailLength int
	q           chan bool
	l           sync.Mutex
	c           chan command
	looptime    time.Duration
	Paused      bool
}

func (o *Orrery) Particles() []*Particle {
	o.l.Lock()
	defer o.l.Unlock()

	r := make([]*Particle, len(o.particles))
	copy(r, o.particles)

	return r
}

func (p Particle) String() string {
	return fmt.Sprintf(`T: %0.2f R:%.2f, M:%.2f, Pos:%s, Vel:%s`, p.T, p.R, p.M, p.Pos, p.Vel)
}

func (p *Particle) move(trailLength int) {
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

func (p *Particle) applyForce(f vector.V3, s float64) {
	p.Vel = p.Vel.Add(f.Scaled(s / p.M))
}

type collision int

const (
	TOTAL collision = iota
	PARTIAL
	NONE
)

func (p *Particle) collide(px *Particle) collision {
	if p == px {
		panic(`can't collide with myself!`)
	}

	if p.M == 0 || px.M == 0 {
		panic(`colliding a particle with zero mass!`)
	}

	p.L.Lock()
	defer p.L.Unlock()

	px.L.Lock()
	defer px.L.Unlock()

	d := p.Pos.Distance(px.Pos)
	if d > p.R+px.R {
		return NONE
	}

	CR := 0.5

	a1 := 2 * px.M / (p.M + px.M)
	v1 := px.Vel.Sub(p.Vel)

	a2 := 2 * p.M / (p.M + px.M)
	v2 := p.Vel.Sub(px.Vel)

	p.applyForce(v1.Normalized(), a1*CR)
	px.applyForce(v2.Normalized(), a2*CR)

	p.T += (a1 * (1 - CR)) / p.M
	px.T += (a2 * (1 - CR)) / px.M

	if d < math.Max(p.R, px.R) {
		return TOTAL
	}

	return PARTIAL
}

func (p *Particle) interactGravity(px *Particle) {
	if px == p {
		panic(`can't gravitationally interact with myself!`)
	}

	p.L.Lock()
	defer p.L.Unlock()

	px.L.Lock()
	defer px.L.Unlock()

	if p.M == 0 || px.M == 0 {
		return
	}

	// G := 6.67 * math.Pow(10, -11)
	G := float64(0.5)

	v := px.Pos.Sub(p.Pos)

	d := math.Max(1, v.Magnitude())

	M := p.M + px.M
	a := (G * M) / (d * d)

	v = v.Normalized()

	p.applyForce(v, a)
	px.applyForce(v, -a)
}

func (o *Orrery) loadUniverse() {
	fh, err := os.Open("universe.json")
	if err != nil {
		log.Printf(`can't open universe.json: %s`, err)
		return
	}
	defer fh.Close()

	d := json.NewDecoder(fh)

	pl := []Particle{}
	err = d.Decode(pl)
	if err != nil {
		log.Printf(`can't decode universe: %s`, err)
		return
	}

	o.l.Lock()
	defer o.l.Unlock()
	o.particles = []*Particle{}
	for _, p := range pl {
		o.particles = append(o.particles, &p)
	}
}

func (o *Orrery) storeUniverse() {
	fname := "universe.json"

	fh, err := os.Create(fname)
	if err != nil {
		log.Fatalf(`can't create %s: %s`, fname, err)
	}
	defer fh.Close()

	e := json.NewEncoder(fh)
	o.l.Lock()
	defer o.l.Unlock()
	err = e.Encode(o.particles)
	if err != nil {
		log.Fatalf(`can't encode universe: %s`, err)
	}
	log.Printf(`dumped universe to %s`, fname)
}

func (o *Orrery) loop() {
	/* XXX: Use barnes-hut simulation for less processing time: O(n^2) -> O(n log n)
	   - https://en.wikipedia.org/wiki/Barnes%E2%80%93Hut_simulation
	*/

	for {
		t_start := time.Now()

		select {
		case c := <-o.c:
			switch c := c.(type) {
			case CommandSpawnParticle:
				if c.M == 0 {
					c.M = 2
				}
				o.particles = append(o.particles, newParticle(c.M, c.Pos, vector.V3{}))
			case CommandSpawnVolume:
				rn := func(r float64) float64 {
					return (rand.Float64() - 0.5) * r
				}

				for i := 0; i < 10; i++ {
					px := vector.V3{c.Pos.X + rn(300), c.Pos.Y + rn(300), c.Pos.Z + rn(300)}
					m := 2.0
					o.particles = append(o.particles, newParticle(m, px, vector.V3{}))
				}
			case CommandPause:
				o.Paused = !o.Paused
			case CommandLoad:
				o.loadUniverse()
			case CommandStore:
				o.storeUniverse()
			default:
				panic(fmt.Sprintf(`unknown orrery command: %T %v`, c, c))
			}
		default:
		}

		if o.Paused {
			time.Sleep(o.looptime)
			continue
		}

		pchan := make(chan [2]*Particle)
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

		o.l.Lock()
		for i, p := range o.particles {
			for _, px := range o.particles[i+1:] {
				wg.Add(1)
				pchan <- [2]*Particle{p, px}
			}
		}
		wg.Wait()
		close(pchan)

		for _, p := range o.particles {
			p.move(o.trailLength)
		}

		// Check for collisions
		garbage := make(map[*Particle]bool)
		for i := 0; i < len(o.particles); i++ {
			p := o.particles[i]
			if garbage[p] {
				continue
			}
			for _, px := range o.particles[i+1:] {
				if garbage[px] {
					continue
				}
				if p.collide(px) == TOTAL {
					/*
					// Merge p and px
					posn := p.Pos.Add(p.Pos.Sub(px.Pos).Scaled(1.0 / 2))
					mn := p.M + px.M
					veln := p.Vel.Scaled(1/p.M).Add(px.Vel.Scaled(1/px.M)).Scaled(mn)
					// TODO: calculate new average temperature from old masses and new mass
					o.particles = append(o.particles, newParticle(mn, posn, veln))
					// Marg p and px for garbage collection
					garbage[p] = true
					garbage[px] = true
					// Restart outer loop to re-check for new collisions
					// XXX: restarting may add additional velocity for new collisions.
					i = 0
					break
					*/
				}
			}
		}

		if len(garbage) > 0 {
			nl := []*Particle{}
			for _, p := range o.particles {
				if !garbage[p] {
					nl = append(nl, p)
				}
			}
			o.particles = nl
		}
		o.l.Unlock()

		t_sleep := o.looptime.Nanoseconds() - time.Since(t_start).Nanoseconds()
		if t_sleep > 0 {
			time.Sleep(time.Duration(t_sleep) * time.Nanosecond)
		}
	}
}

func (o *Orrery) QueueCommand(c command) {
	o.c <- c
}

func newParticle(mass float64, pos vector.V3, vel vector.V3) *Particle {
	return &Particle{
		T: 0, M: mass, R: math.Pow(mass, 1.0/3),
		Vel: vel, Pos: pos,
	}
}

func New() *Orrery {
	o := &Orrery{
		Paused: true,
		trailLength: 20,
		looptime:    5 * time.Millisecond,

		q: make(chan bool),
		c: make(chan command, 20),
/*
		particles:   []*Particle{
			newParticle(5.972*10e2, vector.V3{}, vector.V3{}),
			newParticle(7.346*10e1, vector.V3{3.88*10e1, 0, 0}, vector.V3{0, 0.2, 0}),
		},
		*/
	}

	go o.loop()

	return o
}
