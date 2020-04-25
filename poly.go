package png2svg

import (
	"fmt"
	"image"
	"image/color"
)

// POLYGON
type polygon struct {
	points   map[int]map[int]bool
	process  chan map[string]int
	checked  map[int]map[int]bool
	colorMap map[int]map[int]color.Color
	color    color.Color
	start    map[string]int
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
		p.checkPoint(up)
	}

	if _, ok := p.colorMap[down["x"]][down["y"]]; ok {
		p.checkPoint(down)
	}

	if _, ok := p.colorMap[left["x"]][left["y"]]; ok {
		p.checkPoint(left)
	}

	if _, ok := p.colorMap[right["x"]][right["y"]]; ok {
		p.checkPoint(right)
	}
}

func (p *polygon) checkPoint(start map[string]int) {
	x := start["x"]
	y := start["y"]

	checked := p.checked[x][y]

	if checked {
		// already checked, do nothing.
	} else {

		colorAtPoint := p.colorMap[x][y]
		if colorsMatch(colorAtPoint, p.color) {
			// color matches
			// do stuff
			p.points[x][y] = true
			p.process <- start

		} else {
			// color does not match
			// do other stuff
			p.points[x][y] = false
		}

		p.checked[x][y] = true
	}
}

func (p *polygon) TotalPoints() string {
	return fmt.Sprintf("Total points: %d", len(p.points))
}

func newPolygon(startX, startY int, check map[int]map[int]bool, colors map[int]map[int]color.Color) *polygon {
	poly := &polygon{
		start: map[string]int{
			"x": startX,
			"y": startY,
		},
		checked:  check,
		colorMap: colors,
		color:    colors[startX][startY],
		process:  make(chan map[string]int, 1),
	}
	poly.process <- poly.start
	return poly
}

// POLYGON MANAGER
type PolygonManager struct {
	baseImage   image.Image
	imageHeight int
	imageWidth  int
	polygons    []*polygon
	covered     map[int]map[int]bool
	colorMap    map[int]map[int]color.Color
}

func (pm *PolygonManager) setDimensions() {
	pm.imageWidth = pm.baseImage.Bounds().Max.X
	pm.imageHeight = pm.baseImage.Bounds().Max.Y
}

func (pm *PolygonManager) initialize() {
	pm.setDimensions()

	covered := map[int]map[int]bool{}
	colors := map[int]map[int]color.Color{}

	for x := 0; x < pm.imageWidth; {
		for y := 0; y < pm.imageHeight; {
			colors[x][y] = pm.baseImage.At(x, y)
			covered[x][y] = false
		}
	}

	pm.covered = covered
	pm.colorMap = colors
}

// GENERAL
func NewPolygonManager(img image.Image) *PolygonManager {
	pm := &PolygonManager{
		baseImage: img,
	}

	pm.initialize()

	return pm
}

// UTILITY
func colorsMatch(first, second color.Color) bool {

	r1, g1, b1, a1 := first.RGBA()
	r2, g2, b2, a2 := second.RGBA()

	if r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2 {
		return true
	}

	return false
}
