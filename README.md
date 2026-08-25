# imago

Image algorithms and color math for Go, standard library only.

```go
m := pix.FromImage(src)                    // premultiply into floats
m, err := dpid.Resize(m, width, height, 1) // any scale/* package
out := m.NRGBA()                           // un-premultiply, quantize
```

Packages:

- `scale/{box,resample,dpid,perceptual,l0,contentadaptive}` downscaling
- `pix` float image, `chroma` color spaces, `chroma/nrgba` color helpers
- `filter`, `pixelart`, `retarget/seamcarve`, `metric/ssim`, `enhance/sketch`

Each package documents its own usage and cites its paper.

MIT licensed.
