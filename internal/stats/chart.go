package stats

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vittolewerissa/hbt/internal/shared/ui"
)

// BarChart renders a horizontal bar chart
type BarChart struct {
	Width     int
	MaxValue  float64
	BarChar   rune
	EmptyChar rune
}

// NewBarChart creates a new bar chart
func NewBarChart(width int) *BarChart {
	return &BarChart{
		Width:     width,
		MaxValue:  100,
		BarChar:   '█',
		EmptyChar: '░',
	}
}

// Render renders a single bar
func (c *BarChart) Render(value float64, label string) string {
	if c.MaxValue <= 0 {
		c.MaxValue = 100
	}

	// Calculate bar width
	ratio := value / c.MaxValue
	if ratio > 1 {
		ratio = 1
	}
	if ratio < 0 {
		ratio = 0
	}

	barWidth := c.Width - len(label) - 10 // Reserve space for label and percentage
	if barWidth < 5 {
		barWidth = 5
	}

	filledWidth := int(float64(barWidth) * ratio)
	emptyWidth := barWidth - filledWidth

	// Build the bar
	filled := strings.Repeat(string(c.BarChar), filledWidth)
	empty := strings.Repeat(string(c.EmptyChar), emptyWidth)

	// Color the bar based on value
	var barStyle lipgloss.Style
	switch {
	case ratio >= 0.8:
		barStyle = lipgloss.NewStyle().Foreground(ui.Success)
	case ratio >= 0.5:
		barStyle = lipgloss.NewStyle().Foreground(ui.Warning)
	default:
		barStyle = lipgloss.NewStyle().Foreground(ui.Danger)
	}

	bar := barStyle.Render(filled) + ui.MutedText.Render(empty)

	// Format percentage
	pctStr := fmt.Sprintf("%.0f%%", value)
	pct := lipgloss.NewStyle().Width(5).Align(lipgloss.Right).Render(pctStr)

	return ui.MutedText.Render(label) + " " + bar + " " + pct
}

// Sparkline renders a sparkline chart
type Sparkline struct {
	Width int
}

// SparkChars are the characters used for sparklines
var SparkChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// NewSparkline creates a new sparkline
func NewSparkline(width int) *Sparkline {
	return &Sparkline{Width: width}
}

// Render renders the sparkline from values
func (s *Sparkline) Render(values []float64) string {
	if len(values) == 0 {
		return ""
	}

	// Find min and max
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Handle edge case where all values are the same
	if max == min {
		max = min + 1
	}

	// Map values to sparkline characters
	var result strings.Builder
	charCount := len(SparkChars)

	// Take last Width values if we have more
	start := 0
	if len(values) > s.Width {
		start = len(values) - s.Width
	}

	for i := start; i < len(values); i++ {
		v := values[i]
		// Normalize to 0-1 range
		normalized := (v - min) / (max - min)
		// Map to character index
		idx := int(normalized * float64(charCount-1))
		if idx >= charCount {
			idx = charCount - 1
		}
		if idx < 0 {
			idx = 0
		}

		// Color based on value relative to max
		char := string(SparkChars[idx])
		ratio := v / max
		var style lipgloss.Style
		switch {
		case ratio >= 0.8:
			style = lipgloss.NewStyle().Foreground(ui.Success)
		case ratio >= 0.5:
			style = lipgloss.NewStyle().Foreground(ui.Warning)
		default:
			style = lipgloss.NewStyle().Foreground(ui.Danger)
		}

		result.WriteString(style.Render(char))
	}

	return result.String()
}
