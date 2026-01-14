package model

import "time"

// Category groups related habits together
type Category struct {
	ID        int64
	Name      string
	Color     string // Hex color code (e.g., "#FF5733")
	CreatedAt time.Time
}

// Default category colors for quick selection
var DefaultColors = []string{
	"#FF6B6B", // Red
	"#4ECDC4", // Teal
	"#45B7D1", // Blue
	"#96CEB4", // Green
	"#FFEAA7", // Yellow
	"#DDA0DD", // Plum
	"#98D8C8", // Mint
	"#F7DC6F", // Gold
}
