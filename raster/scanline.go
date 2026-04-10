package raster

import (
	"github.com/soypat/lefevre"
)

// ScanlineRasterizer is a CPU-based scanline rasterizer.
// Internal scratch buffers are reused across calls for heapless operation after warmup.
type ScanlineRasterizer struct {
	// scratch buffers reused across Rasterize calls.
	edges    []edge
	scanline []float32
}

type edge struct {
	x0, y0, x1, y1 float32
}

// Rasterize fills buf with 8-bit alpha coverage for the given outline segments.
func (r *ScanlineRasterizer) Rasterize(buf []byte, width, height, stride int, segments []lefevre.Segment, scale, xoff, yoff float32) {
	if width <= 0 || height <= 0 || len(buf) < stride*height {
		return
	}
	// Clear the target region.
	for y := range height {
		row := buf[y*stride : y*stride+width]
		for i := range row {
			row[i] = 0
		}
	}

	// Build edge list from segments.
	r.edges = r.edges[:0]
	var cx, cy float32 // current point
	for _, seg := range segments {
		sx := float32(seg.X)*scale + xoff
		sy := float32(seg.Y)*scale + yoff
		switch seg.Op {
		case lefevre.SegmentMoveTo:
			cx, cy = sx, sy
		case lefevre.SegmentLineTo:
			r.addEdge(cx, cy, sx, sy)
			cx, cy = sx, sy
		case lefevre.SegmentQuadTo:
			scx := float32(seg.Cx)*scale + xoff
			scy := float32(seg.Cy)*scale + yoff
			r.flattenQuad(cx, cy, scx, scy, sx, sy)
			cx, cy = sx, sy
		case lefevre.SegmentClose:
			// Close is implicit in contour — no edge needed unless we track move-to.
		}
	}

	// Scanline rasterization using non-zero winding rule.
	if cap(r.scanline) < width {
		r.scanline = make([]float32, width)
	}
	r.scanline = r.scanline[:width]

	for y := range height {
		// Clear scanline.
		for i := range r.scanline {
			r.scanline[i] = 0
		}

		fy := float32(y) + 0.5 // sample at pixel center
		for _, e := range r.edges {
			ey0, ey1 := e.y0, e.y1
			if ey0 > ey1 {
				ey0, ey1 = ey1, ey0
			}
			if fy < ey0 || fy >= ey1 {
				continue
			}
			// Compute x intersection.
			t := (fy - e.y0) / (e.y1 - e.y0)
			ix := e.x0 + t*(e.x1-e.x0)

			// Determine winding direction: +1 if edge goes down, -1 if up.
			dir := float32(1)
			if e.y0 > e.y1 {
				dir = -1
			}

			// Distribute coverage across the scanline.
			xi := int(ix)
			if xi >= 0 && xi < width {
				r.scanline[xi] += dir
			}
		}

		// Accumulate winding and convert to coverage.
		row := buf[y*stride : y*stride+width]
		var accum float32
		for x := range width {
			accum += r.scanline[x]
			coverage := accum
			if coverage < 0 {
				coverage = -coverage
			}
			if coverage > 1 {
				coverage = 1
			}
			row[x] = byte(coverage * 255)
		}
	}
}

func (r *ScanlineRasterizer) addEdge(x0, y0, x1, y1 float32) {
	if y0 == y1 {
		return // horizontal edges don't contribute to scanline crossings
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
	if dx*dx+dy*dy < 0.25 { // half-pixel tolerance
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
