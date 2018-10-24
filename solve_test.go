package main

import (
	"bufio"
	"math"
	"strings"
	"testing"
)

type testcomm struct {
	solution Solution
}

func (t *testcomm) send(r Solution) Money {
	t.solution = r
	return r.totalCost
}
func (t *testcomm) done() {
}

func eq(f1, f2 Flight) bool {
	if f1.From != f2.From {
		return false
	}
	if f1.To != f2.To {
		return false
	}
	if f1.FromArea != f2.FromArea {
		return false
	}
	if f1.ToArea != f2.ToArea {
		return false
	}
	if f1.Day != f2.Day {
		return false
	}
	if f1.Cost != f2.Cost {
		return false
	}
	return true
}

func TestSolve(t *testing.T) {
	input := `3 ASD
Green
ASD
Red
SKT
Blue
MXT GDO
ASD MXT 1 50
ASD GDO 1 10
SKT ASD 0 30
MXT SKT 2 20
GDO SKT 2 90
`
	/*ASD := City(0)
	SKT := City(1)
	MXT := City(2)
	GDO := City(3)
	G := Area(0)
	R := Area(1)
	B := Area(2)

	p := readInput(bufio.NewScanner(strings.NewReader(input)))
	flights := []Flight{
		{ASD, MXT, G, B, Day(1), 50, 0, 0.0},
		{ASD, GDO, G, B, Day(1), 10, 0, 0.0},
		{SKT, ASD, R, G, Day(0), 30, 0, 0.0},
		{SKT, ASD, R, G, Day(1), 30, 0, 0.0},
		{SKT, ASD, R, G, Day(2), 30, 0, 0.0},
		{MXT, SKT, B, R, Day(2), 20, 0, 0.0},
		{GDO, SKT, B, R, Day(2), 90, 0, 0.0},
	}
	expected := []*Flight{
		&Flight{0, 1},
	}*/
	p := readInput(bufio.NewScanner(strings.NewReader(input)))
	g := Greedy{p.indices, math.MaxInt32}
	c := &testcomm{}
	g.Solve(c, p)
	printSolution(c.solution, p)
	if c.solution.totalCost != 100 {
		t.Fatalf("sample test cost %v != 100", c.solution.totalCost)
	}
}
