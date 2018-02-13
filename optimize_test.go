package main

import (
	"testing"

	"github.com/Workiva/stretchr/assert"
)

func TestReflectPoint(t *testing.T) {
	center := &Point{
		Dims:  3,
		Terms: []float64{0, 2, 0},
	}
	subject := &Point{
		Dims:  3,
		Terms: []float64{0, 1, 0},
	}
	expected := &Point{
		Dims:  3,
		Terms: []float64{0, 3, 0},
	}
	assert.Equal(t, expected, ReflectPoint(center, subject))
}

func TestComputeCentroid(t *testing.T) {
	points := []*Point{{
		Dims:  2,
		Terms: []float64{1, 1},
	}, {
		Dims:  2,
		Terms: []float64{2, 3},
	}, {
		Dims:  2,
		Terms: []float64{10, 11},
	}}

	expected := &Point{
		Dims:  2,
		Terms: []float64{13.0 / 3.0, 5},
	}
	assert.Equal(t, expected, ComputeCentroid(points...))
}

func TestDrawSimplex(t *testing.T) {
	points := []*Point{{
		Dims:  2,
		Terms: []float64{0, 0},
	}, {
		Dims:  2,
		Terms: []float64{10, 20},
	}, {
		Dims:  2,
		Terms: []float64{20, 10},
	}}
	s := NewSimplex(2)
	s.Points = points
	drawSimplex(s)
}

func TestImproveSimplex(t *testing.T) {
	points := []*Point{{
		Dims:  2,
		Terms: []float64{0, 0},
	}, {
		Dims:  2,
		Terms: []float64{10, 20},
	}, {
		Dims:  2,
		Terms: []float64{20, 10},
	}}
	s := NewSimplex(2)
	evals := []float64{10, 20, 30}
	for i, p := range points {
		s.SetPoint(p, evals[i])
	}

	// If cost is better than all of the existing points,
	// point should be inserted as first evaluation
	add1 := &Point{
		Dims:  2,
		Terms: []float64{20, 20},
	}
	s.Improve(add1, 5)

	expected := append([]*Point{add1}, points[:2]...)
	assert.Equal(t, expected, s.Points)

	// Cost in the middle => point should be inserted in the middle.
	// Current costs = 5, 10, 20
	add2 := &Point{
		Dims:  2,
		Terms: []float64{-1, -2},
	}
	s.Improve(add2, 7)
	expected = []*Point{add1, add2, points[0]}
	assert.Equal(t, expected, s.Points)

	// Cost at the end => point should be inserted at the end.
	// Current costs = 5, 7, 10
	add3 := &Point{
		Dims:  2,
		Terms: []float64{100, 200},
	}
	s.Improve(add3, 9)
	expected = []*Point{add1, add2, add3}
	assert.Equal(t, expected, s.Points)

	// Cost higher than any existing evaluation should panic
	assert.Panics(t, func() {
		s.Improve(&Point{Dims: 2, Terms: []float64{10, 20}}, 100)
	})
}
