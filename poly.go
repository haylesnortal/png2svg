package png2svg

import (
	//"encoding/json"
	//"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"sort"
	"sync"
)

// POLYGON
type polygon struct {
	points   map[int]map[int]bool
	process  chan map[string]int
	checked  map[int]map[int]bool
	colorMap map[int]map[int]color.Color
	color    color.Color
	tolerance int
	start    map[string]int
	counterin int
	counterout int
}

func (p *polygon) checkAdjacent(start map[string]int) {
	x := start["x"]
	y := start["y"]

	up := map[string]int{
		"x": x,
		"y": y - 1,
	}

	down := map[string]int{
		"x": x,
		"y": y + 1,
	}

	left := map[string]int{
		"x": x - 1,
		"y": y,
	}

	right := map[string]int{
		"x": x + 1,
		"y": y,
	}

	if _, ok := p.colorMap[up["x"]][up["y"]]; ok {
		//log.Print("up")
		p.checkPoint(up)
	}

	if _, ok := p.colorMap[down["x"]][down["y"]]; ok {
		//log.Print("down")
		p.checkPoint(down)
	}

	if _, ok := p.colorMap[left["x"]][left["y"]]; ok {
		//log.Print("left")
		p.checkPoint(left)
	}

	if _, ok := p.colorMap[right["x"]][right["y"]]; ok {
		//log.Print("right")
		p.checkPoint(right)
	}
}

func (p *polygon) checkPoint(start map[string]int) {
	x := start["x"]
	y := start["y"]

	checked := p.checked[x][y]

	if checked {
		// already checked, do nothing.
		//log.Print("already checked")
	} else {

		colorAtPoint := p.colorMap[x][y]
		if colorsMatch(colorAtPoint, p.color, p.tolerance) {
			// color matches
			//log.Print("colors match")
			if p.points[x] == nil {
				p.points[x] = make(map[int]bool)
			}
			p.points[x][y] = true
			p.push(start)

		} else {
			// color does not match
			//log.Print("colors don't match")
			if p.points[x] == nil {
				p.points[x] = make(map[int]bool)
			}
			p.points[x][y] = false
		}

		p.checked[x][y] = true
		//log.Print("updated checked")
	}
}

func (p *polygon) processPoints() {

	//log.Print("processPoints()")
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func(p *polygon, wg *sync.WaitGroup) {

		//log.Print("goroutine")

		for p.inchan() > 0 {
			//log.Print("> 0")
			todo := p.pop()
			p.checkPoint(todo)
			p.checkAdjacent(todo)
		}

		wg.Done()

	}(p, wg)
	
	wg.Wait()
}

func (p *polygon) Points() []map[string]int {

	var pts []map[string]int

	for xindex, xval := range p.points {
		for yindex, yval := range xval {
			if yval {
				add := map[string]int{
					"x": xindex,
					"y": yindex,
				}
				pts = append(pts, add)
			}
		} 
	}

	return pts
}

func (p *polygon) PointsCount() int {
	output := 0

	for _, xval := range p.points {
		for _, yval := range xval {
			if yval {
				output++
			}
		} 
	}

	return output
}

func newPolygon(startX, startY int, check map[int]map[int]bool, colors map[int]map[int]color.Color, tolerance int) *polygon {
	poly := &polygon{
		start: map[string]int{
			"x": startX,
			"y": startY,
		},
		checked:  check,
		colorMap: colors,
		color:    colors[startX][startY],
		process:  make(chan map[string]int, 1000000),
		points: make(map[int]map[int]bool),
		counterin: 0,
		counterout: 0,
		tolerance: tolerance,
	}
	poly.push(poly.start)
	return poly
}

func (p *polygon) push(i map[string]int) {
	//log.Print("push()")
	p.counterin = p.counterin +1
	p.process <- i
}

func (p *polygon) pop() map[string]int {
	//log.Print("pop()")
	p.counterout = p.counterout +1
	return <- p.process
}

func (p *polygon) classification() string {
	c := "unknown"

	if p.PointsCount() == 1 {
		c = "singlePixel"
		return c
	}

	xmap := make(map[int][]int)
	ymap := make(map[int][]int)

	for xindex, yvals := range p.points {
		for yindex, val := range yvals {
			if val {
				if xmap[xindex] == nil {
					xmap[xindex] = []int{}
				}
				xmap[xindex] = append(xmap[xindex], yindex)

				if ymap[yindex] == nil {
					ymap[yindex] = []int{}
				}
				ymap[yindex] = append(ymap[yindex], xindex)
			}
		}
	}

	xconsec := make(map[int]map[int][]int)
	yconsec := make(map[int]map[int][]int)

	for key,val := range xmap {
		xconsec[key] = consecutiveNumbers(val)
	}

	for key,val := range ymap {
		yconsec[key] = consecutiveNumbers(val)
	}

	if len(xconsec) == 1 || len(yconsec) == 1 {
		c = "singleLine"
	}

	if len(xconsec) > 1 && len(yconsec) > 1 {
		c = "polygon"
	}

	//xjson,_ := json.MarshalIndent(xconsec, "", "  ")
	//yjson,_ := json.MarshalIndent(yconsec, "", "  ")

	//perm = fmt.Sprintf("X: \n%s\n\nY:\n%s\n", xjson, yjson)

	//log.Print(perm)

	return c
}

func (p *polygon) inchan() int {
	in := p.counterin - p.counterout
	//log.Printf("in channel: %d", in)
	return in
}

// POLYGON MANAGER
type PolygonManager struct {
	baseImage   image.Image
	tolerance int
	imageHeight int
	imageWidth  int
	polygons    []*polygon
	singlePixels []*polygon
	singleLines []*polygon
	covered     map[int]map[int]bool
	colCovered	map[int]bool
	colorMap    map[int]map[int]color.Color
}

func (pm *PolygonManager) setDimensions() {
	pm.imageWidth = pm.baseImage.Bounds().Max.X
	log.Printf("Set width to: %d", pm.imageWidth)
	pm.imageHeight = pm.baseImage.Bounds().Max.Y
	log.Printf("Set height to: %d", pm.imageHeight)
}

func (pm *PolygonManager) initialize() {
	pm.setDimensions()

	covered := make(map[int]map[int]bool)
	colCovered := make(map[int]bool)
	colors := make(map[int]map[int]color.Color)

	for x := 0; x < pm.imageWidth; {

		//log.Printf("Column %d of %d", x, pm.imageWidth)

		colCovered[x] = false

		for y := 0; y < pm.imageHeight; {

			if covered[x] == nil {
				covered[x] = make(map[int]bool)
			}

			if colors[x] == nil {
				colors[x] = make(map[int]color.Color)
			}

			colors[x][y] = pm.baseImage.At(x, y)
			covered[x][y] = false

			y++
		}

		x++
	}

	pm.covered = covered
	pm.colCovered = colCovered
	pm.colorMap = colors
}

func (pm *PolygonManager) ImageHeight() int {
	return pm.imageHeight
}

func (pm *PolygonManager) ImageWidth() int {
	return pm.imageWidth
}

func (pm *PolygonManager) Polygons() []*polygon {
	return pm.polygons
}

func (pm *PolygonManager) SingleLines() []*polygon {
	return pm.singleLines
}

func (pm *PolygonManager) SinglePixels() []*polygon {
	return pm.singlePixels
}

func (pm *PolygonManager) CreatePolygons() error {

	next := pm.NextPixel()

	log.Printf("Creating polygons...")

	//polycounter := 0

	for (next["x"] >= 0 && next["y"] >= 0) {

		//polycounter++
		//log.Printf("Polycounter: %d", polycounter)

		poly := newPolygon(next["x"], next["y"], pm.covered, pm.colorMap, pm.tolerance)
		poly.processPoints()
		//log.Printf("Total points: %d", len(poly.points))
		
		polygonMatchedPoints := 0
		for xindex, ymap := range poly.points {
			for yindex, val := range ymap {
				if val {
					pm.covered[xindex][yindex] = true
					polygonMatchedPoints++
				}
			}
			pm.colCovered[xindex] = columnComplete(pm.covered[xindex])
		}

		if polygonMatchedPoints != 0 {
			class := poly.classification()
			switch class {
				case "singlePixel":
					pm.singlePixels = append(pm.singlePixels, poly)
				case "singleLine":
					pm.singleLines = append(pm.singleLines, poly)
				case "polygon":
					pm.polygons = append(pm.polygons, poly)
				default:
					log.Print("Something went wrong.")
			}
			
		}

		next = pm.NextPixel()
	}

	log.Print("No more pixels.")

	return nil
}

func (pm *PolygonManager) NextPixel() map[string]int {
	
	output := map[string]int{
		"x": -1,
		"y": -1,
	}

	for x,c1 := range pm.colCovered {
		if !c1 {
			output["x"] = x
			for y,c2 := range pm.covered[x] {
				if !c2 {
					output["y"] = y
				}
			}
		}
	}

	//log.Printf("X: %d Y: %d", output["x"], output["y"])

	return output
}

// GENERAL
func NewPolygonManager(img image.Image, tolerance int) *PolygonManager {
	pm := &PolygonManager{
		baseImage: img,
		tolerance: tolerance,
	}

	pm.initialize()

	return pm
}

// UTILITY
func colorsMatch(first, second color.Color, tolerance int) bool {

	r1, g1, b1, a1 := first.RGBA()
	r2, g2, b2, a2 := second.RGBA()

	if withinRange(r1, r2, tolerance) && withinRange(g1, g2, tolerance) && withinRange(b1, b2, tolerance) && withinRange(a1, a2, tolerance) {
		return true
	}

	return false
}

func withinRange(a, b uint32, tolerance int) bool {
	
	if a==b {
		return true
	}
	
	toleranceFloat := float64(tolerance)
	aFloat := float64(a)
	bFloat := float64(b)
	absdiff := math.Abs((aFloat-bFloat))
	output := absdiff <= toleranceFloat

	return output
}

func columnComplete(col map[int]bool) bool {

	output := true

	for _,v := range col {
		if v == false {
			return false
		}
	}

	return output
}

func consecutiveNumbers(n []int) map[int][]int {
	sorted := sort.IntSlice(n)
	sorted.Sort()

	output := make(map[int][]int)
	runCount := 1
	last := -1

	for index,val := range sorted {
		if index == 0 {
			last = val
			newSlice := []int{}
			newSlice = append(newSlice, last)
			output[runCount] = newSlice
		} else {
			if val == last+1 {
				output[runCount] = append(output[runCount], val) 
				last = val
			} else {
				runCount++
				last = val
				newSlice := []int{}
				newSlice = append(newSlice, last)
				output[runCount] = newSlice
			}
		}
	}

	return output
}