# imago

Image algorithms and color math for Go. Stdlib only.

```go
import (
    "github.com/loov/imago/pix"
    "github.com/loov/imago/scale/dpid"
)

m := pix.FromImage(src)                    // premultiply into floats
m, err := dpid.Resize(m, width, height, 1) // filter
out := m.NRGBA()                           // un-premultiply, quantize
```

Conversions are never hidden inside an algorithm: each package takes the
representation it actually computes on and returns the same.

- Float algorithms (`scale/*`, `metric/ssim`) take `*pix.Image`, premultiplied
  RGBA in whatever encoding you give them. Wrap with `(*pix.Image).Linearize` /
  `Delinearize` to filter in linear light. Exception: `scale/contentadaptive`
  interprets its input as sRGB (it works in CIELAB) — pass encoded values.
- Exact 8-bit algorithms (`pixelart`, `retarget/seamcarve`, `enhance/sketch`)
  take `*image.NRGBA`.

Outputs are at origin (0,0); inputs honor non-zero `Bounds().Min`.

## Packages

### Foundation

| Package | What |
|---|---|
| `chroma` | Color: float64 spaces `SRGB`, `RGB` (linear), `XYZ`, `Oklab`/`OkLCh`, `Lab`/`LCh`, `Luv`, `HSL`, `HSV` (star topology, D65, hue in turns, no clamping); `Oklab`/`Lab` `Distance`, `OkLCh.Clamp` gamut mapping. |
| `chroma/nrgba` | Per-frame float32 toolkit for `color.NRGBA`: `RGB`/`RGBA`/`HSL`/`HSLA` (float32), `RGB8`/`Gray8`, named colors, `Hex`/`Parse`/`String`, `Floats`, `Mix`, `Lerp`, `Over`, `MulAlpha`, `Premultiply`, LUT-backed `Linear` (with `Lerp`, `Over`, `Luminance`), `Contrast`/`TextOn`, theme ops `Lighten`/`Darken`/`Saturate`/`Shift`/`Gray` via OkLCh, `Ramp`. |
| `pix` | Premultiplied float RGBA image: `FromImage`, `NRGBA`, `Linearize`, `Delinearize`. The currency of the float algorithms. |
| `filter` | Byte-plane filters: separable 3-tap `Blur`, `Median`, `Erode`. |

### Scaling — `scale/…`

| Package | Algorithm |
|---|---|
| `box` | Area average with exact fractional coverage. The baseline. |
| `resample` | Separable resampling up or down: `Lanczos3`, `CatmullRom`, `MitchellNetravali`, `Mitchell(b, c)`. |
| `dpid` | Rapid, Detail-Preserving Image Downscaling [3]. `lambda = 0` is box. |
| `l0` | L0-regularized downscaling in the spirit of [4]; reconstructed from the L0 framework, not a reference implementation. |
| `perceptual` | Perceptually Based Downscaling of Images [2]. |
| `contentadaptive` | Content-Adaptive Image Downscaling [1]. |

### Other

| Package | Algorithm |
|---|---|
| `pixelart` | `Scale2x`, `Scale3x`, `XBR2x` (level 1). |
| `retarget/seamcarve` | Seam carving [5], shrink only. |
| `metric/ssim` | `SSIM` and `MSSSIM` on luma [6, 7]. |
| `enhance/sketch` | Sketchure [8]: flattens uneven lighting on photographed sketches. |

## References

1. Kopf, Shamir, Peers. "Content-Adaptive Image Downscaling." *ACM TOG* 32(6), 2013. [doi:10.1145/2508363.2508370](https://doi.org/10.1145/2508363.2508370)
2. Öztireli, Gross. "Perceptually Based Downscaling of Images." *ACM TOG* 34(4), 2015. [doi:10.1145/2766891](https://doi.org/10.1145/2766891)
3. Weber, Waechter, Amende, Magnor, Goesele. "Rapid, Detail-Preserving Image Downscaling." *ACM TOG* 35(6), 2016. [doi:10.1145/2980179.2980239](https://doi.org/10.1145/2980179.2980239)
4. Liu, He, Lau, Heng. "L0-Regularized Image Downscaling." *IEEE TIP* 27(3), 2018. [doi:10.1109/TIP.2017.2772838](https://doi.org/10.1109/TIP.2017.2772838)
5. Avidan, Shamir. "Seam Carving for Content-Aware Image Resizing." *ACM TOG* 26(3), 2007. [doi:10.1145/1275808.1276390](https://doi.org/10.1145/1275808.1276390)
6. Wang, Bovik, Sheikh, Simoncelli. "Image Quality Assessment: From Error Visibility to Structural Similarity." *IEEE TIP* 13(4), 2004. [doi:10.1109/TIP.2003.819861](https://doi.org/10.1109/TIP.2003.819861)
7. Wang, Simoncelli, Bovik. "Multiscale Structural Similarity for Image Quality Assessment." *Asilomar* 2003. [doi:10.1109/ACSSC.2003.1292216](https://doi.org/10.1109/ACSSC.2003.1292216)
8. Elbre. Sketchure. https://github.com/loov/sketchure
