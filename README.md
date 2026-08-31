# imago

Image algorithms and color math for Go. The library packages use only the standard library.

```go
m := pix.FromImage(src)                    // premultiply into floats
m, err := dpid.Resize(m, width, height, 1) // any scale/* package
out := m.NRGBA()                           // un-premultiply, quantize
```

Packages:

- `scale/{box,resample,dpid,perceptual,l0,contentadaptive}` downscaling
- `scale/pixelate` downscaling with a palette, `quant` palettes, `dither` ordered dithering
- `pix` float image, `chroma` color spaces, `chroma/nrgba` color helpers
- `filter`, `pixelart`, `retarget/seamcarve`, `metric/ssim`, `enhance/sketch`

Each package documents its own usage and cites its paper.

There is also a command line tool:

```
go install github.com/loov/imago/cmd/imago@latest
imago resize dpid --width 64 input.png output.png
```

MIT licensed.
