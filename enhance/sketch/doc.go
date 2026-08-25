// Package sketch cleans up photographed or scanned pencil sketches by
// flattening uneven paper lighting to a uniform white, a port of
// https://github.com/loov/sketchure (cleanup.ByBase).
//
// The zero Options uses sketchure's defaults: desaturate, whiteness 100,
// line width of one twentieth of the image width.
//
//	out, err := sketch.Clean(src, sketch.Options{})
//
// Set LineWidth to the thickest stroke in pixels when the default swallows
// detail, and KeepColor to preserve colored pencil.
package sketch
