package main

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/gonum/stat"
)

const (
	terminateThreshold = 0.01
	maxIters           = 200
	expandCoeff        = 2
	contractCoeff      = 0.5
	shrinkCoeff        = 0.5
)

type Point struct {
	Dims  int
	Terms []float64
}

func NewPoint(dims int) *Point {
	return &Point{
		Dims:  dims,
		Terms: make([]float64, dims),
	}
}

type Simplex struct {
	Points         []*Point
	Dimension      int
	Evaluations    []float64
	initialized    bool
	numInitialized int
}

func NewSimplex(dim int) *Simplex {
	return &Simplex{
		Points:      make([]*Point, 0),
		Evaluations: make([]float64, 0),
		Dimension:   dim,
	}
}

func ComputeCentroid(points ...*Point) *Point {
	sum := SumPoints(points...)
	return scalePoint(sum, 1/(float64)(len(points)))
}

// StdDev returns the standard deviation of the Simplex's evaluated values
func (s *Simplex) StdDev() float64 {
	return stat.StdDev(s.Evaluations, nil)
}

// Improve "improves" a simplex by replacing its worst
// value with the given value
func (s *Simplex) Improve(p *Point, value float64) {
	i := sort.Search(len(s.Evaluations[0:len(s.Evaluations)-1]),
		func(i int) bool { return s.Evaluations[i] > value })
	if i == len(s.Evaluations) {
		panic(`Improve: provided value is worse than all existing values`)
	}

	// Prevent another slice allocation
	// Do not copy the last element because it is the
	// "worst" and will be trimmed
	fmt.Printf("improve at i %+v\n", i)
	copy(s.Points[i+1:], s.Points[i:len(s.Points)-1])
	fmt.Printf("s.Evaluations: %+v\n", s.Evaluations)
	copy(s.Evaluations[i+1:], s.Evaluations[i:len(s.Evaluations)-1])
	fmt.Printf("s.Evaluations: %+v\n", s.Evaluations)
	s.Points[i] = p
	s.Evaluations[i] = value
}

func (s *Simplex) Cost() float64 {
	cost := 0.0
	for _, e := range s.Evaluations {
		cost += e
	}
	return cost
}

func (s *Simplex) SetPoint(p *Point, value float64) {
	i := sort.Search(len(s.Evaluations),
		func(i int) bool { return value < s.Evaluations[i] })
	if s.numInitialized < s.Dimension+1 {
		// make room for new value
		s.Evaluations = append(s.Evaluations, 0)
		s.Points = append(s.Points, &Point{})
		copy(s.Points[i+1:], s.Points[i:len(s.Points)])
		copy(s.Evaluations[i+1:], s.Evaluations[i:len(s.Evaluations)])
		s.Points[i] = p
		s.Evaluations[i] = value
		s.numInitialized++
	}

	s.Points[i] = p
	s.Evaluations[i] = value
}

func SumPoints(points ...*Point) *Point {
	if len(points) == 0 {
		panic(`SumPoints: no points to sum`)
	}
	acc := &Point{
		Dims:  points[0].Dims,
		Terms: make([]float64, points[0].Dims),
	}
	for d := 0; d < acc.Dims; d++ {
		for _, p := range points {
			acc.Terms[d] += p.Terms[d]
		}
	}
	return acc
}

func scalePoint(p *Point, scalar float64) *Point {
	ret := &Point{
		Dims:  p.Dims,
		Terms: make([]float64, p.Dims),
	}
	for d := 0; d < p.Dims; d++ {
		ret.Terms[d] = p.Terms[d] * scalar
	}
	return ret
}

func ReflectPoint(center, p *Point) *Point {
	scaled := scalePoint(center, 2)
	negated := scalePoint(p, -1)
	return SumPoints(scaled, negated)
}

func ContractPoint(center, p *Point) *Point {
	negated := scalePoint(center, -1)
	sum := scalePoint(SumPoints(p, negated), contractCoeff)
	return SumPoints(center, sum)
}

func shouldTerminate(s *Simplex) bool {
	return s.StdDev() < terminateThreshold
}

func Optimize(eval func(p *Point) float64) {
	dims := 2
	points := initPoints(dims, dims+1)
	simplex := NewSimplex(2)
	for _, p := range points {
		simplex.SetPoint(p, eval(p))
	}
	numIters := 0
	for {
		fmt.Printf("Cost: %+v\n", simplex.Cost())
		numIters++
		if numIters > maxIters || shouldTerminate(simplex) {
			finalVals := `{`
			for _, p := range simplex.Points {
				finalVals += fmt.Sprintf(`%v`, p) + `, `
			}
			finalVals += `}`
			fmt.Printf("final values are %+v at %+v\n", simplex.Evaluations, finalVals)
			return
		}
		centroid := ComputeCentroid(simplex.Points...)
		reflected := ReflectPoint(centroid, simplex.Points[len(simplex.Points)-1])
		fmt.Printf("reflected %+v\n", reflected)
		// if reflected is better than the second worst point,
		// but not better than the best, obtain new simplex which
		// includes the reflected point
		reflectedEval := eval(reflected)
		if reflectedEval < simplex.Evaluations[simplex.Dimension] &&
			reflectedEval > simplex.Evaluations[0] {

			simplex.Improve(reflected, reflectedEval)
			continue
		}
		if reflectedEval < simplex.Evaluations[0] {
			// reflected point is the best so far. Expand
			fmt.Printf("expanding\n")
			negatedCentroid := scalePoint(centroid, -1)
			expanded := SumPoints(centroid, scalePoint(SumPoints(reflected, negatedCentroid), expandCoeff))
			expandedEval := eval(expanded)
			if expandedEval < reflectedEval {
				fmt.Printf("use expanded val %+v\n", expanded)
				simplex.Improve(expanded, expandedEval)
			} else {
				simplex.Improve(expanded, reflectedEval)
			}
			continue
		}
		contracted := ContractPoint(centroid, simplex.Points[len(simplex.Points)-1])
		contractedEval := eval(contracted)
		if contractedEval < simplex.Evaluations[len(simplex.Points)-1] {
			simplex.Improve(contracted, contractedEval)
			continue
		}
		// Shrink the Simplex
		best := simplex.Points[0]
		for i := range simplex.Points[1:] {
			negated := scalePoint(simplex.Points[i], -1)
			shrunk := scalePoint(SumPoints(negated, best),
				shrinkCoeff)
			p := SumPoints(best, shrunk)
			simplex.Points[i] = p
			simplex.Evaluations[i] = eval(p)
		}
	}
}

func main() {
	evalFunc := func(p *Point) float64 {
		//sum := 0.0
		//for _, v := range p.Terms {
		//	sum += v
		//}
		return 10 - p.Terms[0]
		//return sum
	}
	Optimize(evalFunc)
}

func initPoints(dim, count int) []*Point {
	points := make([]*Point, count)
	for i := 0; i < count; i++ {
		points[i] = NewPoint(dim)
		for d := 0; d < dim; d++ {
			r := rand.Float64() * 10
			points[i].Terms[d] = r
		}
	}
	return points
}
