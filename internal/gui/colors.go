package gui

// TagColor holds a palette entry with a hex color string for CSS/HTML.
type TagColor struct {
	Hex string
}

// PaletteSize is the number of colors in the rotating palette.
const PaletteSize = 32

// Palette contains 32 visually distinct colors for window tags.
// First 8 are Apple HIG system colors; the rest extend across hue bands.
var Palette = [PaletteSize]TagColor{
	// ── Primary 8 (Apple HIG) ──
	{Hex: "#FF453A"}, //  0 Red
	{Hex: "#FF9F0A"}, //  1 Orange
	{Hex: "#FFD60A"}, //  2 Yellow
	{Hex: "#30D158"}, //  3 Green
	{Hex: "#40C8E0"}, //  4 Teal
	{Hex: "#0A84FF"}, //  5 Blue
	{Hex: "#BF5AF2"}, //  6 Purple
	{Hex: "#FF375F"}, //  7 Pink
	// ── Extended set ──
	{Hex: "#FF6B6B"}, //  8 Coral
	{Hex: "#E07C24"}, //  9 Burnt Orange
	{Hex: "#A8D830"}, // 10 Lime
	{Hex: "#2EB086"}, // 11 Emerald
	{Hex: "#00A3CC"}, // 12 Cerulean
	{Hex: "#5856D6"}, // 13 Indigo
	{Hex: "#E040A0"}, // 14 Magenta
	{Hex: "#FF8FA0"}, // 15 Rose
	{Hex: "#C44040"}, // 16 Brick
	{Hex: "#CC8800"}, // 17 Amber
	{Hex: "#88B830"}, // 18 Olive Green
	{Hex: "#10A060"}, // 19 Jade
	{Hex: "#3090B0"}, // 20 Steel Blue
	{Hex: "#4060E0"}, // 21 Cobalt
	{Hex: "#9040C0"}, // 22 Violet
	{Hex: "#D05080"}, // 23 Raspberry
	{Hex: "#E08060"}, // 24 Salmon
	{Hex: "#B09030"}, // 25 Brass
	{Hex: "#60A830"}, // 26 Grass
	{Hex: "#20C0A0"}, // 27 Turquoise
	{Hex: "#50A0D0"}, // 28 Sky
	{Hex: "#7070E0"}, // 29 Periwinkle
	{Hex: "#A060D0"}, // 30 Lavender
	{Hex: "#D06090"}, // 31 Peony
}
