package main

import (
	"bufio"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var seed = rand.New(rand.NewSource(time.Now().UnixNano()))
var problem Problem

/* Notes:
 * - more identical flights with different prices can appear - filter during input reading?
 *   - we cannot assume any order of input (available testing data are sorted by day or by src, dst, day)
 * - simulated anealing seems to rock best in last challenge
 * - ending in the same area, not city
 * - index of the first day is 1, 0 has special meaning (flight occuring on every day)
 */

/* TODO:
 * - solution engine
 * - search for better solutions for whole time limit
 */

const MAX_CITIES int = 300
const MAX_AREAS int = 300
const MAX_DAYS int = 300
const MAX_FLIGHTS int = 27000000

type Day uint16
type City uint16
type Area uint16
type Money uint32

type Flight struct {
	From      City
	To        City
	FromArea  Area
	ToArea    Area
	Day       Day
	Cost      Money
	Heuristic Money
	Penalty   float64
}

func p(arr []string, i int) string {
	if len(arr) > i {
		return arr[i]
	} else {
		return fmt.Sprintf("%v", i)
	}
}

func (f *Flight) String() string {
	return fmt.Sprintf("{[%v/%v]->[%v/%v]d%v:$%v}",
		p(problem.areaLookup.indexToName, int(f.FromArea)),
		p(problem.cityLookup.indexToName, int(f.From)),
		p(problem.areaLookup.indexToName, int(f.ToArea)),
		p(problem.cityLookup.indexToName, int(f.To)), f.Day, f.Cost)
}

type comm interface {
	send(r Solution) Money
	done()
	current() Solution
}
type solutionComm struct {
	mutex       *sync.Mutex
	best        Solution
	searchedAll chan bool
	timeout     <-chan time.Time
}

func NewComm(timeout <-chan time.Time) *solutionComm {
	initBest := Solution{}
	initBest.totalCost = math.MaxInt32
	return &solutionComm{
		&sync.Mutex{},
		initBest,
		make(chan bool),
		timeout,
	}
}
func (c *solutionComm) current() Solution {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	flights := make([]*Flight, len(c.best.flights))
	copy(flights, c.best.flights)
	return Solution{flights, c.best.totalCost}
}
func (c *solutionComm) send(r Solution) Money {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if bullshit(r) {
		panic("bullshit")
		return c.best.totalCost
	}
	bestCost := c.best.totalCost
	if bestCost < r.totalCost {
		return bestCost
	}

	flights := make([]*Flight, len(r.flights))
	copy(flights, r.flights)
	sort.Sort(byDay(flights))
	c.best = Solution{flights, r.totalCost}
	return r.totalCost
}
func (c *solutionComm) done() {
	c.searchedAll <- true
}
func (c *solutionComm) wait() {
	select {
	case <-c.searchedAll:
		return
	case <-c.timeout:
		return
	}
}

type partial struct {
	flights []*Flight
	visited []bool
	n       int
	cost    Money
}

func (p *partial) solution() Solution {
	return Solution{p.flights, p.cost}
}
func (p *partial) roundtrip() bool {
	ff := p.flights[0]
	lf := p.lastFlight()
	for _, ok := range p.visited {
		if !ok {
			return false
		}
	}
	isHome := lf.ToArea == ff.FromArea
	return isHome
}
func (p *partial) fly(f *Flight) {
	p.visited[int(f.FromArea)] = true
	p.flights = append(p.flights, f)
	p.cost += f.Cost
}
func (p *partial) hasVisited(a Area) bool {
	return p.visited[a]
}
func (p *partial) lastFlight() *Flight {
	return p.flights[len(p.flights)-1]
}
func (p *partial) backtrack() {
	f := p.flights[len(p.flights)-1]
	p.visited[int(f.FromArea)] = false
	p.flights = p.flights[0 : len(p.flights)-1]
	p.cost -= f.Cost
}

func order(i, j int) (int, int) {
	if i < j {
		return i, j
	}
	return j, i
}

type sa struct {
	graph  FlightIndices
	areaDb AreaDb
}

func (d *sa) run(comm comm) {
	current := comm.current()
	cost := current.totalCost
	best := cost
	flights := current.flights
	g := problem.indices.fromDayTo
	areadb := problem.areaDb
	//temp := 0
	maxCitySwap, maxAreaSwap := len(flights)-2, len(flights)-1
	for {
		newBest := false
		//don't swap first and last city
		//i, j := randomFlightSwap(maxCitySwap)
		i, j := bestFlightSwap(current, g, maxCitySwap)
		ok, newCost := swapFlights(current, g, i, j, false)
		if ok {
			if best > newCost {
				best, newBest = newCost, true
				current.totalCost = newCost
				swapFlights(current, g, i, j, true)
			} else {
				//TODO do this with some probability
				current.totalCost = newCost
				swapFlights(current, g, i, j, true)
			}
		}
		//don't swap first city but can swap last city
		fi, ci := randomAreaSwap(maxAreaSwap, flights, areadb)
		ok, newCost = swapInArea(current, g, fi, ci, false)
		if ok {
			if best > newCost {
				best, newBest = newCost, true
				current.totalCost = newCost
				swapInArea(current, g, fi, ci, true)
			} else {
				//TODO do this with some probability
				current.totalCost = newCost
				swapInArea(current, g, fi, ci, true)
			}
		}
		if newBest {
			fmt.Fprintln(os.Stderr, "sa new solution", best)
			best = comm.send(Solution{flights, best})
		}
	}
}

/*
0 ---- 1 ---- 2 ---- 4
A      B      C      A
a->b   b->c   c->d
fiPrev fi
a->X   X->c   c->d
giPrev gi
*/
func swapInArea(s Solution, g Graph, i int, x City, really bool) (bool, Money) {
	if i == -1 {
		return false, 0
	}
	flights := s.flights
	prevI := i - 1
	fiPrev := flights[prevI]
	giPrev := g.get(fiPrev.From, fiPrev.Day, x)
	fi := flights[i]
	gi := g.get(x, fi.Day, fi.To)
	if giPrev != nil && gi != nil {
		oldCost := fiPrev.Cost + fi.Cost
		newCost := giPrev.Cost + gi.Cost
		if really {
			flights[prevI] = giPrev
			flights[i] = gi
		}
		return true, s.totalCost - oldCost + newCost
	}
	return false, 0
}

/*
0 ---- 1 ---- 2 ---- 3 ---- 4
A      B      C      D      A
a->b   b->c   c->d   d->a
fiPrev fi     fjPrev fj
a->d   d->c   c->b   b->a
giPrev gi     gjPrev gj
*/
func swapFlights(s Solution, g Graph, i, j int, really bool) (bool, Money) {
	if i == -1 || j == -1 {
		return false, 0
	}
	flights := s.flights
	prevI := i - 1
	prevJ := j - 1
	fiPrev := flights[prevI]
	fjPrev := flights[prevJ]
	giPrev := g.get(fiPrev.From, fiPrev.Day, fjPrev.To)
	gjPrev := g.get(fjPrev.From, fjPrev.Day, fiPrev.To)
	fi := flights[i]
	fj := flights[j]
	gi := g.get(fj.From, fi.Day, fi.To)
	gj := g.get(fi.From, fj.Day, fj.To)
	if giPrev != nil && gjPrev != nil && gi != nil && gj != nil {
		oldCost := fiPrev.Cost + fi.Cost + fjPrev.Cost + fj.Cost
		newCost := giPrev.Cost + gi.Cost + gjPrev.Cost + gj.Cost
		if really {
			flights[prevI] = giPrev
			flights[i] = gi
			flights[prevJ] = gjPrev
			flights[j] = gj
		}
		return true, s.totalCost - oldCost + newCost
	}
	return false, 0
}

/*****************************************************************************/
/* SA heuristics                                                             */
/*****************************************************************************/

//TODO: cache this function and flag when already tried something
//maybe use the total solution cost and inteligently flight pointers
func bestFlightSwap(s Solution, g Graph, max int) (int, int) {
	bi, bj, best := -1, -1, Money(math.MaxInt32)
	maxi := max - 1
	for i := 1; i <= maxi; i++ {
		for j := i + 1; j <= max; j++ {
			ok, newCost := swapFlights(s, g, i, j, false)
			if ok && best > newCost {
				best, bi, bj = newCost, i, j
			}
		}
	}
	return bi, bj
}

//TODO: cache this function and flag when already tried something
//maybe use the total solution cost and inteligently flight pointers
func bestAreaSwap(s Solution, g Graph, max int, flights []*Flight, areadb AreaDb) (int, City) {
	bfi, bci, best := -1, City(0), Money(math.MaxInt32)
	for fi := 1; fi <= max; fi++ {
		from := flights[fi].From
		a := areadb.cityToArea[from]
		area := areadb.areaToCities[a]
		for _, ci := range area {
			ok, newCost := swapInArea(s, g, fi, ci, false)
			if ok && best > newCost {
				best, bfi, bci = newCost, fi, ci
			}
		}
	}
	return bfi, bci
}

//TODO: could use some heuristics instead of random maybe
func randomFlightSwap(n int) (int, int) {
	i := seed.Intn(n)
	j := seed.Intn(n)
	for ; j == i; j = seed.Intn(n) {
	}
	return order(i+1, j+1)
}

//TODO: could use some heuristics instead of random maybe
func randomAreaSwap(n int, flights []*Flight, areadb AreaDb) (int, City) {
	fi := seed.Intn(n-1) + 1
	from := flights[fi].From
	a := areadb.cityToArea[from]
	area := areadb.areaToCities[a]
	if len(area) < 2 {
		return -1, 0
	}
	ci := seed.Intn(len(area))
	max := 5
	for ; area[ci] == from && max > 0; ci = seed.Intn(len(area)) {
		max--
	}
	if max == 0 {
		return -1, 0
	}

	return fi, City(area[ci])
}

/*****************************************************************************/
/* Greedy                                                                    */
/*****************************************************************************/

type Greedy struct {
	graph       FlightIndices
	currentBest Money
	finished    bool
	endOnFirst  bool
}

func (d *Greedy) dfs(comm comm, partial *partial) {
	if d.finished {
		return
	}
	if partial.cost > d.currentBest {
		return
	}
	if partial.roundtrip() {
		d.currentBest = comm.send(partial.solution())
		d.finished = d.currentBest == partial.cost && d.endOnFirst
		return
	}
	lf := partial.lastFlight()
	if partial.hasVisited(lf.ToArea) {
		return
	}
	var dst []*Flight
	if len(partial.flights) == partial.n-1 {
		if d.graph.areaDayCost[lf.ToArea] == nil {
			return
		}
		if d.graph.areaDayCost[lf.ToArea][lf.Day+1] == nil {
			return
		}
		dst = d.graph.areaDayCost[lf.ToArea][lf.Day+1]
	} else {
		if d.graph.cityDayCost[lf.To] == nil {
			return
		}
		if d.graph.cityDayCost[lf.To][lf.Day+1] == nil {
			return
		}
		dst = d.graph.cityDayCost[lf.To][lf.Day+1]
	}
	for _, f := range dst {
		partial.fly(f)
		d.dfs(comm, partial)
		partial.backtrack()
	}
}
func (d Greedy) Solve(comm comm) {
	if len(problem.cityLookup.indexToName) > 10 {
		d.endOnFirst = true
	}
	flights := make([]*Flight, 0, problem.length)
	visited := make([]bool, problem.length, problem.length)
	partial := partial{flights, visited, problem.length, 0}

	dst := d.graph.cityDayCost[0][1]
	for _, f := range dst {
		partial.fly(f)
		d.dfs(comm, &partial)
		partial.backtrack()
	}

	if !d.endOnFirst {
		comm.done()
	} else {
		sa := sa{}
		sa.run(comm)
	}
}

func NewSolution(flights []*Flight) Solution {
	sort.Sort(byDay(flights))
	return Solution{flights, cost(flights)}
}

type Solution struct {
	flights   []*Flight
	totalCost Money
}

type LookupA struct {
	nameToIndex map[string]Area
	indexToName []string
}

type LookupC struct {
	nameToIndex map[string]City
	indexToName []string
}
type Graph [][][]*Flight

func (g *Graph) get(f City, d Day, t City) *Flight {
	if (*g)[f] == nil {
		return nil
	}
	if (*g)[f][d] == nil {
		return nil
	}
	return (*g)[f][d][t]
}

type FlightIndices struct {
	areaDayCost [][][]*Flight // sorted by cost
	cityDayCost [][][]*Flight // sorted by cost
	fromDayTo   Graph         // not sorted
	//dayArea     [][][]*Flight
	//dayCity     [][][]*Flight
}

type AreaDb struct {
	cityToArea   map[City]Area
	areaToCities map[Area][]City
}

type Problem struct {
	flights []*Flight
	indices FlightIndices
	//areas []Area
	areaDb     AreaDb
	areaLookup LookupA
	cityLookup LookupC
	start      City
	goal       Area
	length     int
	timeLimit  time.Duration
}

type byCost []*Flight

func (f byCost) Len() int {
	return len(f)
}
func (f byCost) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}
func (f byCost) Less(i, j int) bool {
	return f[i].Cost < f[j].Cost
}

type byDay []*Flight

func (f byDay) Len() int {
	return len(f)
}
func (f byDay) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}
func (f byDay) Less(i, j int) bool {
	return f[i].Day < f[j].Day
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func cityIndex(city string, l *LookupC) City {
	/* get index of city in lookup table or put it in the table and get index */
	ci, found := l.nameToIndex[city]
	if found {
		return ci
	}
	ci = City(len(l.nameToIndex))
	l.nameToIndex[city] = ci
	l.indexToName = append(l.indexToName, city)
	return ci
}

func areaIndex(area string, l *LookupA) Area {
	ai, found := l.nameToIndex[area]
	if found {
		return ai
	}
	ai = Area(len(l.nameToIndex))
	l.nameToIndex[area] = ai
	l.indexToName = append(l.indexToName, area)
	return ai
}

func LastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func flightSplit(s string, r []string) {
	/* Splits lines of input into 4 parts
	   strictly expects format "{3}[A-Z] {3}[A-Z] \d \d"
	   WARNING: no checks are done at all */
	r[0] = s[:3]
	r[1] = s[4:7]
	pos2 := LastIndexByte(s, ' ')
	r[2] = s[8:pos2]
	r[3] = s[pos2+1:]
}

func createIndexAD(slice [][][]*Flight, from Area, day Day, flight *Flight) {
	if slice[from] == nil {
		slice[from] = make([][]*Flight, MAX_DAYS+1)
	}
	if slice[from][day] == nil {
		slice[from][day] = make([]*Flight, 0, MAX_CITIES+1) // is there a max number of flights from a city on a date?
	}
	slice[from][day] = append(slice[from][day], flight)
}

func createIndexCD(slice [][][]*Flight, from City, day Day, flight *Flight) {
	if slice[from] == nil {
		slice[from] = make([][]*Flight, MAX_DAYS+1)
	}
	if slice[from][day] == nil {
		slice[from][day] = make([]*Flight, 0, MAX_CITIES+1) // is there a max number of flights from a city on a date?
	}
	slice[from][day] = append(slice[from][day], flight)
}

func fromDayTo(slice [][][]*Flight, f *Flight) {
	if slice[f.From] == nil {
		slice[f.From] = make([][]*Flight, MAX_DAYS+1)
	}
	if slice[f.From][f.Day] == nil {
		slice[f.From][f.Day] = make([]*Flight, MAX_CITIES)
	}
	if slice[f.From][f.Day][f.To] == nil || slice[f.From][f.Day][f.To].Cost > f.Cost {
		slice[f.From][f.Day][f.To] = f
	}
}

func readInput(stdin *bufio.Scanner) {
	lookupC := &LookupC{make(map[string]City), make([]string, 0, MAX_CITIES)}
	lookupA := &LookupA{make(map[string]Area), make([]string, 0, MAX_AREAS)}
	areaDb := &AreaDb{make(map[City]Area), make(map[Area][]City)}
	flights := make([]*Flight, 0, MAX_FLIGHTS)
	indices := &FlightIndices{make([][][]*Flight, MAX_AREAS),
		make([][][]*Flight, MAX_CITIES),
		make([][][]*Flight, MAX_CITIES),
		//make([][][]*Flight, MAX_DAYS),
		//make([][][]*Flight, MAX_DAYS),
	}
	line := make([]string, 4)

	var src string
	var timeLimit time.Duration
	var length, i int
	var from, to City
	var fromArea, toArea Area
	var day Day
	var cost Money
	// read first line
	if stdin.Scan() {
		firstLine := strings.Split(stdin.Text(), " ")
		src = firstLine[1]
		length, _ = strconv.Atoi(firstLine[0])
		cityIndex(src, lookupC)
	}
	// read areas
	var area string
	var areaId Area
	var cityId City
	cities := make([]string, 0)
	for i := 0; i < length; i++ {
		stdin.Scan()
		area = stdin.Text()
		stdin.Scan()
		cities = strings.Split(stdin.Text(), " ")
		cityIds := make([]City, 0, len(cities))
		areaId = areaIndex(area, lookupA)
		for _, src := range cities {
			cityId = cityIndex(src, lookupC)
			areaDb.cityToArea[cityId] = areaId
			cityIds = append(cityIds, cityId)
		}
		areaDb.areaToCities[areaId] = cityIds

	}
	// read flights
	for stdin.Scan() {
		flightSplit(stdin.Text(), line)
		i, _ = strconv.Atoi(line[2])
		day = Day(i)
		i, _ = strconv.Atoi(line[3])
		cost = Money(i)
		from = cityIndex(line[0], lookupC)
		to = cityIndex(line[1], lookupC)
		//fromArea = areaIndex(line[0], LookupA)
		//toArea = areaIndex(line[1], LookupA)
		fromArea = areaDb.cityToArea[from]
		toArea = areaDb.cityToArea[to]
		if from == City(0) && day != 1 {
			// ignore any flight from src city not on the first day
			// fmt.Fprintln(os.Stderr, "Dropping flight", l)
			continue
		}
		if day == 1 && from != City(0) {
			// also flights originating in different than home city are wasteful
			// fmt.Fprintln(os.Stderr, "Dropping flight", l)
			continue
		}
		if int(day) != 0 && int(day) != length && toArea == areaDb.cityToArea[0] {
			// get rid of flights to final destination on different than last day
			// fmt.Fprintln(os.Stderr, "Dropping", day, from, "->", to)
			continue
		}
		if int(day) == 0 {
			// this flight takes place on every day, we will generate all the flights instead
			for i := 1; i <= length; i++ {
				if toArea == areaDb.cityToArea[0] && i < length {
					// fmt.Fprintln(os.Stderr, "Dropping", i, from, "->", to, length)
					continue
				}
				f := &Flight{from, to, fromArea, toArea, Day(i), cost, 0, 0.0}
				flights = append(flights, f)
				createIndexAD(indices.areaDayCost, fromArea, Day(i), f)
				createIndexCD(indices.cityDayCost, from, Day(i), f)
				fromDayTo(indices.fromDayTo, f)
			}
			continue
		}

		f := &Flight{from, to, fromArea, toArea, day, cost, 0, 0.0}
		flights = append(flights, f)
		createIndexAD(indices.areaDayCost, fromArea, day, f)
		createIndexCD(indices.cityDayCost, from, day, f)
		fromDayTo(indices.fromDayTo, f)

	}
	if length <= 20 {
		timeLimit = 3
	} else if length <= 100 {
		timeLimit = 5
	} else {
		timeLimit = 15
	}

	for _, dayList := range indices.areaDayCost {
		for _, flightList := range dayList {
			sort.Sort(byCost(flightList))
		}
	}

	for _, dayList := range indices.cityDayCost {
		for _, flightList := range dayList {
			sort.Sort(byCost(flightList))
		}
	}

	problem = Problem{flights, *indices, *areaDb, *lookupA, *lookupC,
		City(0), areaDb.cityToArea[City(0)], length, timeLimit}
}

func cost(path []*Flight) Money {
	var cost Money
	for _, f := range path {
		cost += f.Cost
	}
	return cost
}

func printSolution(s Solution) {
	fmt.Println(s.totalCost)
	for i := 0; i < problem.length; i++ {
		fmt.Println(problem.cityLookup.indexToName[s.flights[i].From],
			problem.cityLookup.indexToName[s.flights[i].To],
			i+1,
			s.flights[i].Cost,
		)
	}
}

func bullshit(s Solution) bool {
	length := 0
	prevF := s.flights[0]
	totalCost := Money(prevF.Cost)
	visited := make(map[Area]bool)
	for _, f := range s.flights[1:] {
		totalCost += f.Cost
		if prevF.To != f.From || prevF.Day != (f.Day-1) {
			fmt.Fprintln(os.Stderr, f, "doesnt follow", prevF, "@", length)
			return true
		}
		if visited[f.ToArea] {
			fmt.Fprintln(os.Stderr, f, "tries to revisit area", problem.areaLookup.indexToName[f.ToArea])
			return true
		}
		length += 1
		visited[f.ToArea] = true
		prevF = f
	}
	if totalCost != s.totalCost {
		fmt.Fprintln(os.Stderr, s.totalCost, "!=", totalCost)
		return true
	}
	return false
}

func validateSolution(s Solution) {
	length := 0
	prevF := s.flights[0]
	totalCost := Money(prevF.Cost)
	visited := make(map[Area]bool)
	for _, f := range s.flights[1:] {
		totalCost += f.Cost
		if prevF.To != f.From || prevF.Day != (f.Day-1) {
			fmt.Fprintln(os.Stderr, f, "doesnt follow", prevF, "@", length)
		}
		if visited[f.ToArea] {
			fmt.Fprintln(os.Stderr, f, "tries to revisit", problem.areaLookup.indexToName[f.To])
		}
		length += 1
		visited[f.ToArea] = true
		prevF = f
	}
	if totalCost != s.totalCost {
		fmt.Fprintln(os.Stderr, s.totalCost, "!=", totalCost)
	}
	if length != (problem.length - 1) {
		fmt.Fprintln(os.Stderr, problem.length, "!=", length)
	}
}

func main() {
	start_time := time.Now()
	//defer profile.Start(profile.MemProfile).Stop()
	readInput(bufio.NewScanner(os.Stdin))
	g := Greedy{graph: problem.indices, currentBest: math.MaxInt32}
	timeout := time.After(problem.timeLimit*time.Second - time.Since(start_time) - 45*time.Millisecond)
	c := NewComm(timeout)
	go g.Solve(c)
	c.wait()

	printSolution(c.current())
	validateSolution(c.current())

	fmt.Fprintln(os.Stderr, "Ending after", time.Since(start_time))
}
