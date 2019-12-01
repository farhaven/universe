package vector

import "testing"

func TestV3Normalize(t *testing.T) {
	v := V3{1, 1, 1}.Normalized()
	if v.Magnitude() != 1 {
		t.Errorf(`expected magnitude 1, got %f`, v.Magnitude())
	}
}

func TestPlaneDistance(t *testing.T) {
	norm := V3{0, 0, 1}.Normalized()

	p0 := V3{0, 0, 0}
	pl := Plane{norm, p0}

	if d := pl.Distance(V3{0, 0, 1}); d != 1 {
		t.Errorf(`d=%f`, d)
	}

	if d := pl.Distance(V3{0, 0, 0}); d != 0 {
		t.Errorf(`d=%f`, d)
	}

	if d := pl.Distance(V3{1, 1, 0}); d != 0 {
		t.Errorf(`d=%f`, d)
	}
}
