package handlers

import (
	"fmt"
	"html/template"
	"strings"
)

// DotChartMonthlySVG renders the ReViz-style dot chart for the monthly
// income / cost / expense triple. Values are cents.
func DotChartMonthlySVG(income, cost, expense []int64) template.HTML {
	const (
		w   = 880
		h   = 240
		pl  = 56.0
		pr  = 16.0
		pt  = 24.0
		pb  = 28.0
		max = 32000 * 100 // cents; matches the design's 32k Y max
	)
	innerW := float64(w) - pl - pr
	innerH := float64(h) - pt - pb
	xStep := innerW / 11.0
	xAt := func(i int) float64 { return pl + float64(i)*xStep }
	yAt := func(v int64) float64 {
		if v > max {
			v = max
		}
		return pt + innerH - float64(v)/float64(max)*innerH
	}
	ticks := []int64{0, 8000_00, 16000_00, 24000_00, 32000_00}
	months := []string{"1月", "2月", "3月", "4月", "5月", "6月", "7月", "8月", "9月", "10月", "11月", "12月"}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg class="lg-dotchart" viewBox="0 0 %d %d">`, w, h)

	// gridlines
	b.WriteString(`<g class="grid">`)
	for _, t := range ticks {
		y := yAt(t)
		dash := "2 3"
		if t == 0 {
			dash = ""
		}
		fmt.Fprintf(&b, `<line x1="%g" x2="%g" y1="%g" y2="%g" stroke-dasharray="%s"/>`,
			pl, float64(w)-pr, y, y, dash)
	}
	b.WriteString(`</g>`)

	// Y axis labels
	b.WriteString(`<g class="axis">`)
	for _, t := range ticks {
		label := "0"
		if t > 0 {
			label = fmt.Sprintf("%dk", t/100/1000)
		}
		fmt.Fprintf(&b, `<text x="%g" y="%g" text-anchor="end">%s</text>`,
			pl-10, yAt(t)+4, label)
	}
	b.WriteString(`</g>`)

	// X axis labels
	b.WriteString(`<g class="axis">`)
	for i, m := range months {
		fmt.Fprintf(&b, `<text x="%g" y="%g" text-anchor="middle">%s</text>`,
			xAt(i), float64(h)-8, m)
	}
	b.WriteString(`</g>`)

	series := []struct {
		key, color string
		vals       []int64
	}{
		{"income", "var(--success-500)", income},
		{"cost", "var(--warning-500)", cost},
		{"expense", "var(--danger-500)", expense},
	}

	for _, s := range series {
		// peak value
		var peak int64
		for _, v := range s.vals {
			if v > peak {
				peak = v
			}
		}
		hasAny := peak > 0

		fmt.Fprintf(&b, `<g class="series-%s">`, s.key)
		if hasAny {
			// connecting dotted polyline through every point
			var pts []string
			for i, v := range s.vals {
				pts = append(pts, fmt.Sprintf("%g,%g", xAt(i), yAt(v)))
			}
			fmt.Fprintf(&b,
				`<polyline fill="none" stroke="%s" stroke-width="1" stroke-dasharray="3 3" opacity="0.45" points="%s"/>`,
				s.color, strings.Join(pts, " "))
		}
		for i, v := range s.vals {
			isPeak := v == peak && v > 0
			r := "1.6"
			fillColor := "var(--border-base)"
			if v > 0 {
				fillColor = s.color
				if isPeak {
					r = "5"
				} else {
					r = "3.5"
				}
			}
			if isPeak {
				fmt.Fprintf(&b,
					`<circle cx="%g" cy="%g" r="10" fill="%s" opacity="0.15"/>`,
					xAt(i), yAt(v), s.color)
			}
			fmt.Fprintf(&b, `<circle cx="%g" cy="%g" r="%s" fill="%s"/>`,
				xAt(i), yAt(v), r, fillColor)
			if isPeak {
				fmt.Fprintf(&b,
					`<text x="%g" y="%g" text-anchor="middle" font-family="var(--font-mono)" font-size="10.5" fill="var(--text-strong)" font-weight="500">%dk</text>`,
					xAt(i), yAt(v)-12, v/100/1000)
			}
		}
		b.WriteString(`</g>`)
	}

	b.WriteString(`</svg>`)
	return template.HTML(b.String())
}

// DotChartNetSVG renders the YTD net-profit trend dot chart.
func DotChartNetSVG(net []int64) template.HTML {
	const (
		w  = 400
		h  = 200
		pl = 52.0
		pr = 12.0
		pt = 18.0
		pb = 26.0
	)
	innerW := float64(w) - pl - pr
	innerH := float64(h) - pt - pb
	xStep := innerW / 11.0
	xAt := func(i int) float64 { return pl + float64(i)*xStep }

	var minV, maxV int64
	for _, v := range net {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	if minV == 0 && maxV == 0 {
		maxV = 1
	}
	span := maxV - minV
	if span == 0 {
		span = 1
	}
	yAt := func(v int64) float64 {
		return pt + innerH - float64(v-minV)/float64(span)*innerH
	}
	zeroY := yAt(0)

	// find min index
	minIdx := -1
	for i, v := range net {
		if minV < 0 && v == minV {
			minIdx = i
			break
		}
	}

	var path strings.Builder
	for i, v := range net {
		cmd := "L"
		if i == 0 {
			cmd = "M"
		}
		fmt.Fprintf(&path, "%s%g,%g ", cmd, xAt(i), yAt(v))
	}
	pathStr := strings.TrimSpace(path.String())

	var b strings.Builder
	fmt.Fprintf(&b, `<svg class="lg-dotchart" viewBox="0 0 %d %d" style="height:200px;">`, w, h)
	// zero baseline
	fmt.Fprintf(&b, `<line x1="%g" x2="%g" y1="%g" y2="%g" stroke="var(--border-base)" stroke-width="1"/>`,
		pl, float64(w)-pr, zeroY, zeroY)
	// Y labels — 0 and min if negative
	b.WriteString(`<g class="axis">`)
	fmt.Fprintf(&b, `<text x="%g" y="%g" text-anchor="end">0</text>`, pl-10, zeroY+4)
	if minV < 0 {
		fmt.Fprintf(&b, `<text x="%g" y="%g" text-anchor="end">%dk</text>`, pl-10, yAt(minV)+4, minV/100/1000)
	}
	b.WriteString(`</g>`)
	// X labels — month numbers only (no 月 to keep tight)
	b.WriteString(`<g class="axis">`)
	for i := 0; i < 12; i++ {
		fmt.Fprintf(&b, `<text x="%g" y="%g" text-anchor="middle">%d</text>`, xAt(i), float64(h)-8, i+1)
	}
	b.WriteString(`</g>`)

	// area + line
	areaPath := pathStr + fmt.Sprintf(" L%g,%g L%g,%g Z", xAt(len(net)-1), zeroY, xAt(0), zeroY)
	fmt.Fprintf(&b, `<path d="%s" fill="var(--danger-50)"/>`, areaPath)
	fmt.Fprintf(&b, `<path d="%s" fill="none" stroke="var(--danger-500)" stroke-width="1.5"/>`, pathStr)

	// dots
	for i, v := range net {
		isMin := i == minIdx && v < 0
		r := "2.5"
		fill := "var(--danger-500)"
		if v == 0 {
			fill = "var(--border-base)"
		}
		if isMin {
			r = "4.5"
			fmt.Fprintf(&b, `<circle cx="%g" cy="%g" r="9" fill="var(--danger-500)" opacity="0.18"/>`,
				xAt(i), yAt(v))
		}
		fmt.Fprintf(&b, `<circle cx="%g" cy="%g" r="%s" fill="%s"/>`, xAt(i), yAt(v), r, fill)
		if isMin {
			fmt.Fprintf(&b,
				`<text x="%g" y="%g" text-anchor="middle" font-family="var(--font-mono)" font-size="10.5" fill="var(--danger-700)" font-weight="500">%dk</text>`,
				xAt(i), yAt(v)+22, v/100/1000)
		}
	}

	b.WriteString(`</svg>`)
	return template.HTML(b.String())
}
