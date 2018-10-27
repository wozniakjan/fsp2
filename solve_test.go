package main

import (
	"bufio"
	"math"
	"reflect"
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
func (t *testcomm) current() Solution {
	return t.solution
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

	problem := readInput(bufio.NewScanner(strings.NewReader(input)))
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
	readInput(bufio.NewScanner(strings.NewReader(input)))
	g := Greedy{graph: problem.indices, currentBest: math.MaxInt32}
	c := &testcomm{}
	g.Solve(c)
	printSolution(c.solution)
	if c.solution.totalCost != 100 {
		t.Fatalf("sample test cost %v != 100", c.solution.totalCost)
	}
}

func TestAreaSolve(t *testing.T) {
	input := `3 ASD
Green
ASD TMP
Red
SKT
Blue
MXT GDO
ASD MXT 1 50
ASD GDO 1 10
SKT TMP 0 30
MXT SKT 2 20
GDO SKT 2 90
`
	readInput(bufio.NewScanner(strings.NewReader(input)))
	g := Greedy{graph: problem.indices, currentBest: math.MaxInt32}
	c := &testcomm{}
	g.Solve(c)
	printSolution(c.solution)
	if c.solution.totalCost != 100 {
		t.Fatalf("sample test cost %v != 100", c.solution.totalCost)
	}
}
func mockGraph() Graph {
	g := emptyGraph()
	g[1][1][2] = f(1, 2, 1, 5)
	g[2][2][3] = f(2, 3, 2, 10)
	g[3][3][4] = f(3, 4, 3, 10)
	g[4][4][1] = f(4, 1, 4, 10)

	g[1][1][4] = f(1, 4, 1, 10)
	g[4][2][3] = f(4, 3, 2, 5)
	g[3][3][2] = f(3, 2, 3, 5)
	g[2][4][1] = f(2, 1, 4, 10)

	g[1][1][7] = &Flight{1, 7, 1, 2, 1, 6, 0, 0.0}
	g[7][2][3] = &Flight{7, 3, 2, 3, 2, 6, 0, 0.0}
	return g
}
func emptyGraph() Graph {
	s := 8
	g := Graph{}
	g = make([][][]*Flight, s)
	for i := 0; i < s; i++ {
		g[i] = make([][]*Flight, s)
		for j := 0; j < s; j++ {
			g[i][j] = make([]*Flight, s)
		}
	}
	return g
}
func f(from City, to City, d Day, c Money) *Flight {
	return &Flight{from, to, Area(from), Area(to), d, c, 0, 0.0}
}
func TestGet(t *testing.T) {
	tests := []struct {
		graph    Graph
		from     City
		day      Day
		to       City
		expected *Flight
	}{
		{
			graph:    mockGraph(),
			from:     1,
			day:      1,
			to:       2,
			expected: f(1, 2, 1, 5),
		},
	}
	for ti, test := range tests {
		f := test.graph.get(test.from, test.day, test.to)
		if !reflect.DeepEqual(test.expected, f) {
			t.Fatal(ti, "flight mismatch")
		}
	}
}
func TestSwap(t *testing.T) {
	tests := []struct {
		flights  []*Flight
		graph    Graph
		expected []*Flight
		cost     Money
		i        int
		j        int
		ok       bool
	}{
		{
			flights: []*Flight{
				f(1, 2, 1, 5),
				f(2, 3, 2, 10),
				f(3, 4, 3, 10),
				f(4, 1, 4, 10),
			},
			graph: mockGraph(),
			expected: []*Flight{
				f(1, 4, 1, 10),
				f(4, 3, 2, 5),
				f(3, 2, 3, 5),
				f(2, 1, 4, 10),
			},
			i:    1,
			j:    3,
			cost: 30,
			ok:   true,
		},
		{
			flights: []*Flight{
				f(1, 2, 1, 5),
				f(2, 3, 2, 10),
				f(3, 4, 3, 10),
				f(4, 1, 4, 10),
			},
			graph: emptyGraph(),
			expected: []*Flight{
				f(1, 2, 1, 5),
				f(2, 3, 2, 10),
				f(3, 4, 3, 10),
				f(4, 1, 4, 10),
			},
			i:    1,
			j:    2,
			cost: 0,
			ok:   false,
		},
	}
	for ti, test := range tests {
		cpy := make([]*Flight, len(test.flights))
		copy(cpy, test.flights)
		ok, newCost := swapFlights(Solution{cpy, cost(test.flights)}, test.graph, test.i, test.j, true)
		if newCost != test.cost {
			t.Fatal(ti, "money mismatch")
		}
		if ok != test.ok {
			t.Fatal(ti, "ok mismatch")
		}
		if ok && !reflect.DeepEqual(cpy, test.expected) {
			t.Fatal(ti, "flight mismatch")
		}
		ok, newCost = swapFlights(Solution{cpy, newCost}, test.graph, test.i, test.j, true)
		if ok != test.ok {
			t.Fatal(ti, "back ok mismatch")
		}
		if ok && newCost != cost(test.flights) {
			t.Fatal(ti, "back money mismatch", newCost)
		}
		if ok && !reflect.DeepEqual(cpy, test.flights) {
			t.Fatal(ti, "back flight mismatch")
		}
	}
}

func TestSwapArea(t *testing.T) {
	tests := []struct {
		flights  []*Flight
		graph    Graph
		expected []*Flight
		cost     Money
		i        int
		j        City
		ok       bool
	}{
		{
			flights: []*Flight{
				f(1, 2, 1, 5),
				f(2, 3, 2, 10),
				f(3, 4, 3, 10),
				f(4, 1, 4, 10),
			},
			graph: mockGraph(),
			expected: []*Flight{
				&Flight{1, 7, 1, 2, 1, 6, 0, 0.0},
				&Flight{7, 3, 2, 3, 2, 6, 0, 0.0},
				f(3, 4, 3, 10),
				f(4, 1, 4, 10),
			},
			i:    1,
			j:    7,
			cost: 32,
			ok:   true,
		},
		{
			flights: []*Flight{
				f(1, 2, 1, 5),
				f(2, 3, 2, 10),
				f(3, 4, 3, 10),
				f(4, 1, 4, 10),
			},
			graph: emptyGraph(),
			expected: []*Flight{
				f(1, 2, 1, 5),
				f(2, 3, 2, 10),
				f(3, 4, 3, 10),
				f(4, 1, 4, 10),
			},
			i:    1,
			j:    2,
			cost: 0,
			ok:   false,
		},
	}
	for ti, test := range tests {
		cpy := make([]*Flight, len(test.flights))
		copy(cpy, test.flights)
		ok, newCost := swapInArea(Solution{cpy, cost(test.flights)}, test.graph, test.i, test.j, true)
		if newCost != test.cost {
			t.Fatal(ti, "money mismatch")
		}
		if ok != test.ok {
			t.Fatal(ti, "ok mismatch")
		}
		if ok && !reflect.DeepEqual(cpy, test.expected) {
			t.Fatal(ti, "flight mismatch")
		}
	}
}
