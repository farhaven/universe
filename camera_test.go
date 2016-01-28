package main

import (
	"os"
	"log"
	"testing"
	"./vector"
	"github.com/go-gl-legacy/gl"
)

func TestMain(m *testing.M) {
	if r := gl.Init(); r != 1 {
		log.Fatalf(`can't init GL: %d`, r)
	}
	os.Exit(m.Run())
}

func TestSphereInFrustum(t *testing.T) {
	c := NewCamera(1440, 900, -40, 40, 10)
	c.Update()

	if f := c.SphereInFrustum(vector.V3{0, 0, 0}, 30); f != INTERSECT {
		t.Errorf(`expected INTERSECT, got %s`, f.String())
	}

	if f := c.SphereInFrustum(vector.V3{0, 0, -100}, 1); f != OUTSIDE {
		t.Errorf(`expected OUTSIDE, got %s`, f.String())
	}
}
