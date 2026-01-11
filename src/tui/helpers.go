package tui

import (
	"strings"
)

func cleanStatus(text string) string {
	parts := strings.Fields(text)
	return strings.Join(parts, " ")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func computeUsableWidth(totalWidth int) int {
	if totalWidth == 0 {
		totalWidth = 96
	}
	usableWidth := totalWidth - 1
	if usableWidth < 40 {
		usableWidth = totalWidth
	}
	return usableWidth
}

func computeSubmenuWidths(usableWidth int) (int, int, int) {
	third := usableWidth / 3
	colWidth := third
	tempWidth := third
	brightWidth := usableWidth - colWidth - tempWidth
	if brightWidth < 10 {
		brightWidth = 10
	}
	return colWidth, tempWidth, brightWidth
}

// computeColorCols decides how many color columns to show based on width.
func computeColorCols(width int) int {
	w := width
	if w == 0 {
		w = 96
	}
	usable := w - 6
	colWidth := 12
	cols := usable / colWidth
	if cols < 3 {
		cols = 3
	}
	if cols > 8 {
		cols = 8
	}
	return cols
}

func (m model) colorContentWidth() int {
	usable := computeUsableWidth(m.width)
	colWidth, _, _ := computeSubmenuWidths(usable)
	content := colWidth - 2
	if content < 10 {
		content = colWidth
	}
	return max(content, 10)
}

// buildRange returns count evenly spaced ints between min and max (inclusive).
func buildRange(min, max, count int) []int {
	if count <= 1 {
		return []int{max}
	}
	step := (max - min) / (count - 1)
	if step <= 0 {
		step = 1
	}
	values := make([]int, count)
	for i := 0; i < count; i++ {
		val := min + i*step
		if val > max {
			val = max
		}
		values[i] = val
	}
	return values
}

func clampIndex(val, count int) int {
	if count <= 0 {
		return 0
	}
	if val < 0 {
		return 0
	}
	if val >= count {
		return count - 1
	}
	return val
}

func moveGridIndex(current, total, cols, dRow, dCol int) int {
	if total == 0 || cols <= 0 {
		return 0
	}
	rows := (total + cols - 1) / cols
	r := current / cols
	c := current % cols
	r += dRow
	c += dCol
	if r < 0 {
		r = 0
	}
	if r >= rows {
		r = rows - 1
	}
	if c < 0 {
		c = 0
	}
	if c >= cols {
		c = cols - 1
	}
	idx := r*cols + c
	if idx >= total {
		idx = total - 1
	}
	if idx < 0 {
		idx = 0
	}
	return idx
}

// layoutWords returns number of words per line given max width (with single spaces).
func layoutWords(words []string, maxWidth int) []int {
	if len(words) == 0 {
		return []int{}
	}
	if maxWidth <= 0 {
		maxWidth = 10
	}
	var counts []int
	currentLen := 0
	currentCount := 0
	for _, w := range words {
		wordLen := len(w)
		if currentLen == 0 {
			currentLen = wordLen
			currentCount = 1
			continue
		}
		if currentLen+1+wordLen <= maxWidth {
			currentLen += 1 + wordLen
			currentCount++
		} else {
			counts = append(counts, currentCount)
			currentLen = wordLen
			currentCount = 1
		}
	}
	counts = append(counts, currentCount)
	return counts
}

func moveInLines(current int, counts []int, dRow, dCol int) int {
	if len(counts) == 0 {
		return 0
	}
	line, pos := findLineAndPos(counts, current)
	line += dRow
	if line < 0 {
		line = 0
	}
	if line >= len(counts) {
		line = len(counts) - 1
	}
	pos += dCol
	if pos < 0 {
		pos = 0
	}
	if pos >= counts[line] {
		pos = counts[line] - 1
	}
	return indexFromLinePos(counts, line, pos)
}

func findLineAndPos(counts []int, idx int) (int, int) {
	remaining := idx
	for line, count := range counts {
		if remaining < count {
			return line, remaining
		}
		remaining -= count
	}
	return len(counts) - 1, counts[len(counts)-1] - 1
}

func indexFromLinePos(counts []int, line, pos int) int {
	if line < 0 {
		line = 0
	}
	if line >= len(counts) {
		line = len(counts) - 1
	}
	if pos < 0 {
		pos = 0
	}
	if pos >= counts[line] {
		pos = counts[line] - 1
	}
	idx := 0
	for i := 0; i < line; i++ {
		idx += counts[i]
	}
	idx += pos
	return idx
}

func pickWordIndex(words []string, contentX int) int {
	if len(words) == 0 {
		return 0
	}
	if contentX < 0 {
		contentX = 0
	}
	acc := 0
	for i, w := range words {
		if contentX <= acc+len(w) {
			return i
		}
		acc += len(w) + 1
	}
	return len(words) - 1
}
