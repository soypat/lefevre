package raster

import (
	"github.com/soypat/lefevre"
)

// ScanlineRasterizer is a CPU scanline rasterizer with 4x vertical
// supersampling and fractional-x coverage distribution. Scratch buffers are
// reused across calls for heapless operation after warmup.
type ScanlineRasterizer struct {
	edges    []edge
	scanline []float32
}

type edge struct {
	x0, y0, x1, y1 float32
}

const (
	scanlineSubN    = 4
	scanlineSubStep = float32(1.0) / scanlineSubN
	scanlineWeight  = float32(1.0) / scanlineSubN
)

// Rasterize fills buf with 8-bit alpha coverage for the given outline segments.
func (r *ScanlineRasterizer) Rasterize(buf []byte, width, height, stride int, segments []lefevre.Segment, scale, xoff, yoff float32) {
	if width <= 0 || height <= 0 || len(buf) < stride*height {
		return
	}
	// Clear target region.
	for y := range height {
		row := buf[y*stride : y*stride+width]
		for i := range row {
			row[i] = 0
		}
	}

	// Build edge list from segments.
	r.edges = r.edges[:0]
	var cx, cy, startX, startY float32
	contourStart := true
	for _, seg := range segments {
		sx := float32(seg.X)*scale + xoff
		sy := float32(seg.Y)*scale + yoff
		switch seg.Op {
		case lefevre.SegmentMoveTo:
			cx, cy = sx, sy
			startX, startY = sx, sy
			contourStart = false
		case lefevre.SegmentLineTo:
			r.addEdge(cx, cy, sx, sy)
			cx, cy = sx, sy
		case lefevre.SegmentQuadTo:
			scx := float32(seg.Cx)*scale + xoff
			scy := float32(seg.Cy)*scale + yoff
			r.flattenQuad(cx, cy, scx, scy, sx, sy)
			cx, cy = sx, sy
		case lefevre.SegmentClose:
			// Close is implicit in contour: no edge needed unless we track move-to.
			if !contourStart && (cx != startX || cy != startY) {
				r.addEdge(cx, cy, startX, startY)
			}
			cx, cy = startX, startY
			contourStart = true
		}
	}

	// +1 slot so right-of-pixel delta writes never overflow.
	if cap(r.scanline) < width+1 {
		r.scanline = make([]float32, width+1)
	}
	r.scanline = r.scanline[:width+1]

	for y := range height {
		for i := range r.scanline {
			r.scanline[i] = 0
		}
		fy0 := float32(y)
		for _, e := range r.edges {
			// Determine winding direction: +1 if edge goes down, -1 if up.
			dir := float32(1)
			lo, hi := e.y0, e.y1
			if e.y0 > e.y1 {
				dir = -1
				lo, hi = e.y1, e.y0
			}
			if hi <= fy0 || lo >= fy0+1 {
				continue
			}
			dy := e.y1 - e.y0
			for sub := range scanlineSubN {
				sy := fy0 + (float32(sub)+0.5)*scanlineSubStep
				if sy < lo || sy >= hi {
					continue
				}
				t := (sy - e.y0) / dy
				x := e.x0 + t*(e.x1-e.x0)
				w := scanlineWeight * dir
				if x <= 0 {
					r.scanline[0] += w
					continue
				}
				if x >= float32(width) {
					continue
				}
				xi := int(x)
				frac := x - float32(xi)
				r.scanline[xi] += (1 - frac) * w
				r.scanline[xi+1] += frac * w
			}
		}

		row := buf[y*stride : y*stride+width]
		var accum float32
		for x := range width {
			accum += r.scanline[x]
			c := accum
			if c < 0 {
				c = -c
			}
			if c > 1 {
				c = 1
			}
			row[x] = byte(c * 255)
		}
	}
}

func (r *ScanlineRasterizer) addEdge(x0, y0, x1, y1 float32) {
	if y0 == y1 {
		return
	}
	r.edges = append(r.edges, edge{x0: x0, y0: y0, x1: x1, y1: y1})
}

// flattenQuad recursively flattens a quadratic bezier into line edges.
func (r *ScanlineRasterizer) flattenQuad(x0, y0, cx, cy, x1, y1 float32) {
	// Flatness test: if control point is close enough to the midpoint of the
	// chord, the curve is flat enough to approximate as a line.
	mx := (x0 + x1) / 2
	my := (y0 + y1) / 2
	dx := cx - mx
	dy := cy - my
	// half pixel tolerance.
	if dx*dx+dy*dy < 0.25 {
		r.addEdge(x0, y0, x1, y1)
		return
	}
	// De Casteljau subdivision.
	ax := (x0 + cx) / 2
	ay := (y0 + cy) / 2
	bx := (cx + x1) / 2
	by := (cy + y1) / 2
	abx := (ax + bx) / 2
	aby := (ay + by) / 2
	r.flattenQuad(x0, y0, ax, ay, abx, aby)
	r.flattenQuad(abx, aby, bx, by, x1, y1)
}
