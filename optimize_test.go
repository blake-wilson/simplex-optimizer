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
