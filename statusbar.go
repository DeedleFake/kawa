package main

import "deedles.dev/kawa/geom"

func (server *Server) statusBarBounds(out *Output) geom.Rect[float64] {
	ob := server.outputBounds(out)
	return geom.Rt(ob.Min.X, ob.Min.Y-StatusBarHeight, ob.Max.X, ob.Min.Y)
}
