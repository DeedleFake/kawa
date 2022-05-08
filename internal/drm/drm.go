package drm

//func fourccCode(a, b, c, d uint32) uint32 {
//	return a | (b << 8) | (c << 16) | (d << 24)
//}

const (
	FormatARGB8888 = 'A' | ('R' << 8) | ('2' << 16) | ('4' << 24)
	FormatRGBA8888 = 'R' | ('A' << 8) | ('2' << 16) | ('4' << 24)
)
