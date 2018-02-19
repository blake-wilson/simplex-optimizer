package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"

	"github.com/gonum/stat"
	"github.com/llgcode/draw2d/draw2dimg"
)

const (
	terminateThreshold = 0.01
	maxIters           = 10
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
	i := sort.Search(len(s.Evaluations[0:len(s.Evaluations)]),
		func(i int) bool { return s.Evaluations[i] > value })
	if i == len(s.Evaluations) {
		panic(`Improve: provided value is worse than all existing values`)
	}

	// Prevent another slice allocation
	// Do not copy the last element because it is the
	// "worst" and will be trimmed
	copy(s.Points[i+1:], s.Points[i:len(s.Points)-1])
	copy(s.Evaluations[i+1:], s.Evaluations[i:len(s.Evaluations)-1])
	s.Points[i] = p
	s.Evaluations[i] = value
}

func (s *Simplex) Cost() float64 {
	return s.Evaluations[0]
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

func Optimize(eval func(p *Point) float64) *Simplex {
	dims := 2
	points := initPoints(dims, dims+1)
	simplex := NewSimplex(2)
	file, err := os.Create(`simplex.txt`)
	if err != nil {
		panic(err.Error())
	}
	defer file.Close()
	w := bufio.NewWriter(file)
	file.Sync()

	for _, p := range points {
		simplex.SetPoint(p, eval(p))
	}
	numIters := 0
	for {
		writeSimplex(simplex, w)
		fmt.Printf("Cost: %+v\n", simplex.Cost())
		numIters++
		if numIters > maxIters || shouldTerminate(simplex) {
			finalVals := `{`
			for _, p := range simplex.Points {
				finalVals += fmt.Sprintf(`%v`, p) + `, `
			}
			finalVals += `}`
			fmt.Printf("final values are %+v at %+v\n", simplex.Evaluations, finalVals)
			fmt.Printf("final cost is %+v\n", simplex.Cost())
			break
		}
		centroid := ComputeCentroid(simplex.Points...)
		reflected := ReflectPoint(centroid, simplex.Points[len(simplex.Points)-1])
		// if reflected is better than the second worst point,
		// but not better than the best, obtain new simplex which
		// includes the reflected point
		reflectedEval := eval(reflected)
		if reflectedEval < simplex.Evaluations[simplex.Dimension] &&
			reflectedEval > simplex.Evaluations[0] {
			fmt.Printf("Reflect\n\n")

			simplex.Improve(reflected, reflectedEval)
			continue
		}
		if reflectedEval < simplex.Evaluations[0] {
			// reflected point is the best so far. Expand
			negatedCentroid := scalePoint(centroid, -1)
			expanded := SumPoints(centroid, scalePoint(SumPoints(reflected, negatedCentroid), expandCoeff))
			expandedEval := eval(expanded)
			if expandedEval < reflectedEval {
				simplex.Improve(expanded, expandedEval)
				fmt.Printf("Expand\n\n")

			} else {
				fmt.Printf("Reflect\n\n")
				simplex.Improve(expanded, reflectedEval)
			}
			continue
		}
		contracted := ContractPoint(centroid, simplex.Points[len(simplex.Points)-1])
		contractedEval := eval(contracted)
		if contractedEval < simplex.Evaluations[len(simplex.Points)-1] {
			fmt.Printf("Contract\n\n")
			simplex.Improve(contracted, contractedEval)
			continue
		}
		// Shrink the Simplex
		best := simplex.Points[0]
		for i := range simplex.Points[1:] {
			fmt.Printf("Shrink\n\n")
			negated := scalePoint(simplex.Points[i], -1)
			shrunk := scalePoint(SumPoints(negated, best),
				shrinkCoeff)
			p := SumPoints(best, shrunk)
			simplex.Points[i] = p
			simplex.Evaluations[i] = eval(p)
		}
	}

	w.Flush()
	file.Sync()
	return simplex
}

func main() {
	evalFunc := func(p *Point) float64 {
		//sum := 0.0
		//for _, v := range p.Terms {
		//	sum += v
		//}
		// return 10 - p.Terms[0]

		// ex1
		// return math.Abs(p.Terms[1] - p.Terms[0])

		// ex2
		// return math.Pow(p.Terms[1]-3, 2) + math.Pow(p.Terms[0]-4, 2)

		// ex3
		v := math.Sqrt(math.Pow(p.Terms[0], 2)+
			math.Pow(p.Terms[1], 2)) + math.Nextafter(1.0, 2.0) - 1.0
		return math.Sin(v) / v
		//return sum
	}
	s := Optimize(evalFunc)
	drawSimplex(s)
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

// SubtractMean constructs a new simplex whose
// points have been recentered around 0
func (s *Simplex) SubtractMean() *Simplex {
	averages := make([]float64, s.Dimension)
	for _, p := range s.Points {
		for d := 0; d < s.Dimension; d++ {
			averages[d] += p.Terms[d]
		}
	}
	for i := 0; i < len(averages); i++ {
		averages[i] = averages[i] / float64(len(s.Points))
	}

	s2 := NewSimplex(s.Dimension)
	s2.Points = make([]*Point, len(s.Points))
	for i, p := range s.Points {
		s2.Points[i] = NewPoint(s.Dimension)
		for d := 0; d < s.Dimension; d++ {
			s2.Points[i].Terms[d] = p.Terms[d] - averages[d]
		}
	}
	return s2
}

// TranslateToPositive translates all the coordinates of the given
// Simplex's points to nonnegative values
func (s *Simplex) TranslateToPositive() *Simplex {
	mins := make([]float64, s.Dimension)
	for d := 0; d < len(s.Points[0].Terms); d++ {
		mins[d] = s.Points[0].Terms[d]
	}
	for _, p := range s.Points[1:] {
		for d := 0; d < len(p.Terms); d++ {
			if p.Terms[d] < mins[d] {
				mins[d] = p.Terms[d]
			}
		}
	}
	s2 := NewSimplex(s.Dimension)
	newPoints := make([]*Point, len(s.Points))
	for i, p := range s.Points {
		newPoints[i] = NewPoint(s.Dimension)
		for d := 0; d < len(p.Terms); d++ {
			newPoints[i].Terms[d] = p.Terms[d] - mins[d]
		}
	}
	s2.Points = newPoints
	return s2
}

func writeSimplex(s *Simplex, w *bufio.Writer) {
	// For xi = (xi1, xi2, ..., xin), zi = eval(xi)
	// Print the Simplex in the format
	// Simplex
	// x11,x12,...,x1n,z1
	// x21,x22,...,x2n,z2
	// ..
	// xn1,xn2,...,x(n+1)n, zn+1
	// End
	_, err := w.WriteString("Simplex\n")
	if err != nil {
		panic(err.Error())
	}
	for i, p := range s.Points {
		terms := make([]string, len(p.Terms))
		for j, d := range p.Terms {
			terms[j] = fmt.Sprintf("%f", d)
		}
		terms = append(terms, fmt.Sprintf("%f", s.Evaluations[i]))
		_, err = w.WriteString(strings.Join(terms, `,`) + "\n")
		if err != nil {
			panic(err.Error())
		}
	}
	_, err = w.WriteString("End\n")
	if err != nil {
		panic(err.Error())
	}
}

func drawSimplex(s *Simplex) {

	imgWidth := 850.0
	imgHeight := 850.0
	rect := image.Rect(0, 0, int(imgWidth), int(imgHeight))
	dest := image.NewRGBA(rect)
	gc := draw2dimg.NewGraphicContext(dest)

	// Set some properties
	gc.SetFillColor(color.RGBA{0x44, 0xff, 0x44, 0xff})
	// gc.SetStrokeColor(color.RGBA{0x44, 0x44, 0x44, 0xff})
	gc.SetStrokeColor(color.RGBA{0xff, 0x00, 0x00, 0xff})
	gc.SetLineWidth(5)

	s2 := s.SubtractMean()
	s2 = s2.TranslateToPositive()
	sizeX, sizeY := simplexSize(s)
	pxMult := math.Min(float64(imgWidth/sizeX), float64(imgHeight/sizeY))

	start := translateCoords(s2.Points[0], pxMult)

	colors := []color.RGBA{{
		0x00, 0xff, 0x00, 0xff,
	}, {
		0x00, 0x00, 0xff, 0xff,
	}}
	gc.MoveTo(float64(start.Terms[0]), float64(start.Terms[1]))
	for i, p := range s2.Points[1:] {
		ip := translateCoords(p, pxMult)
		gc.LineTo(float64(ip.Terms[0]), float64(ip.Terms[1]))
		gc.FillStroke()
		gc.MoveTo(float64(ip.Terms[0]), float64(ip.Terms[1]))
		gc.SetStrokeColor(colors[i])
	}
	// Close the loop
	gc.LineTo(float64(start.Terms[0]), float64(start.Terms[1]))
	gc.FillStroke()

}

func writeImage(img *image.Image) {
	f, err := os.Create("image.png")
	if err != nil {
		log.Fatal(err)
	}

	if err := png.Encode(f, *img); err != nil {
		f.Close()
		log.Fatal(err)
	}

	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}

// simplexSize returns the height and width of a 2-D simplex
func simplexSize(s *Simplex) (float64, float64) {
	minX, maxX := s.Points[0].Terms[0], s.Points[0].Terms[0]
	minY, maxY := s.Points[0].Terms[1], s.Points[0].Terms[1]
	for _, p := range s.Points[1:] {
		if p.Terms[0] < minX {
			minX = p.Terms[0]
		}
		if p.Terms[0] > maxX {
			maxX = p.Terms[0]
		}
		if p.Terms[1] < minY {
			minY = p.Terms[1]
		}
		if p.Terms[1] > maxY {
			maxY = p.Terms[1]
		}
	}
	return maxX - minX, maxY - minY
}

func translateCoords(p *Point, stepSize float64) *Point {
	p.Terms[0] *= stepSize
	p.Terms[1] *= stepSize
	imgPoint := NewPoint(2)
	imgPoint.Terms[0] = p.Terms[0]
	imgPoint.Terms[1] = p.Terms[1]
	return imgPoint
}
