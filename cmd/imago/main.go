// Command imago scales images with the algorithms from github.com/loov/imago.
//
//	imago resize dpid --width 64 input.png output.png
//
// Giving only one of --width and --height keeps the aspect ratio. Output
// format follows the output file extension (.png, .jpg, .gif).
package main

import (
	"context"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zeebo/clingy"

	"github.com/loov/imago/pix"
	"github.com/loov/imago/scale/box"
	"github.com/loov/imago/scale/contentadaptive"
	"github.com/loov/imago/scale/dpid"
	"github.com/loov/imago/scale/l0"
	"github.com/loov/imago/scale/perceptual"
	"github.com/loov/imago/scale/pixelate"
	"github.com/loov/imago/scale/resample"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	ok, err := clingy.Environment{Name: "imago"}.Run(ctx, func(cmds clingy.Commands) {
		cmds.Group("resize", "Scale an image down (or up, where supported)", func() {
			cmds.New("box", "Exact area average", &cmdResize{algo: "box"})
			cmds.New("lanczos3", "Separable Lanczos, 3 lobes; up or down", &cmdResize{algo: "lanczos3"})
			cmds.New("catmullrom", "Separable Catmull-Rom; up or down", &cmdResize{algo: "catmullrom"})
			cmds.New("mitchell", "Separable Mitchell-Netravali; up or down", &cmdResize{algo: "mitchell"})
			cmds.New("dpid", "Detail-preserving downscaling", &cmdResize{algo: "dpid", hasLambda: true, defLambda: 1})
			cmds.New("perceptual", "SSIM-optimized downscaling", &cmdResize{algo: "perceptual"})
			cmds.New("l0", "L0-regularized downscaling", &cmdResize{algo: "l0", hasLambda: true, defLambda: l0.DefaultLambda})
			cmds.New("contentadaptive", "Content-adaptive kernels; slow", &cmdResize{algo: "contentadaptive", srgbOnly: true})
			cmds.New("pixelate", "Joint downscale and palette", &cmdResize{algo: "pixelate", srgbOnly: true, hasPalette: true})
		})
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "imago:", err)
	}
	if !ok || err != nil {
		os.Exit(1)
	}
}

type cmdResize struct {
	algo       string
	hasLambda  bool
	defLambda  float64
	hasPalette bool
	srgbOnly   bool // interprets input as sRGB; --linear does not apply

	width, height int
	lambda        float64
	colors        int
	dither        float64
	linear        bool
	quality       int
	input, output string
}

func (c *cmdResize) Setup(params clingy.Parameters) {
	atoi := clingy.Transform(strconv.Atoi)
	parseFloat := clingy.Transform(func(s string) (float64, error) { return strconv.ParseFloat(s, 64) })

	c.width = params.Flag("width", "output width; 0 derives it from --height", 0, atoi).(int)
	c.height = params.Flag("height", "output height; 0 derives it from --width", 0, atoi).(int)
	if c.hasLambda {
		c.lambda = params.Flag("lambda", "detail weight", c.defLambda, parseFloat).(float64)
	}
	if c.hasPalette {
		c.colors = params.Flag("colors", "palette size", 16, atoi).(int)
		c.dither = params.Flag("dither", "dither strength 0..1", 0.0, parseFloat).(float64)
	}
	if !c.srgbOnly {
		c.linear = params.Flag("linear", "filter in linear light", false,
			clingy.Transform(strconv.ParseBool), clingy.Boolean).(bool)
	}
	c.quality = params.Flag("quality", "jpeg output quality", 90, atoi).(int)
	c.input = params.Arg("input", "input image (png, jpg, gif)").(string)
	c.output = params.Arg("output", "output image; format from the extension (png, jpg, gif)").(string)
}

func (c *cmdResize) Execute(ctx context.Context) error {
	f, err := os.Open(c.input)
	if err != nil {
		return err
	}
	src, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		return fmt.Errorf("%s: %w", c.input, err)
	}

	width, height := c.width, c.height
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	switch {
	case width > 0 && height > 0:
	case width > 0:
		height = max((width*sh+sw/2)/sw, 1)
	case height > 0:
		width = max((height*sw+sh/2)/sh, 1)
	default:
		return fmt.Errorf("need --width or --height")
	}

	m := pix.FromImage(src)
	if c.linear {
		m = m.Linearize()
	}

	var out image.Image
	switch c.algo {
	case "pixelate":
		out, err = pixelate.Resize(m, width, height, pixelate.Options{Colors: c.colors, Dither: c.dither})
	default:
		var dst *pix.Image
		switch c.algo {
		case "box":
			dst, err = box.Resize(m, width, height)
		case "lanczos3":
			dst, err = resample.Resize(m, width, height, resample.Lanczos3)
		case "catmullrom":
			dst, err = resample.Resize(m, width, height, resample.CatmullRom)
		case "mitchell":
			dst, err = resample.Resize(m, width, height, resample.MitchellNetravali)
		case "dpid":
			dst, err = dpid.Resize(m, width, height, c.lambda)
		case "perceptual":
			dst, err = perceptual.Resize(m, width, height)
		case "l0":
			dst, err = l0.Resize(m, width, height, c.lambda)
		case "contentadaptive":
			dst, err = contentadaptive.Resize(m, width, height)
		}
		if err == nil {
			if c.linear {
				dst = dst.Delinearize()
			}
			out = dst.NRGBA()
		}
	}
	if err != nil {
		return err
	}

	w, err := os.Create(c.output)
	if err != nil {
		return err
	}
	defer w.Close()
	switch ext := strings.ToLower(filepath.Ext(c.output)); ext {
	case ".png":
		err = png.Encode(w, out)
	case ".jpg", ".jpeg":
		err = jpeg.Encode(w, out, &jpeg.Options{Quality: c.quality})
	case ".gif":
		err = gif.Encode(w, out, nil)
	default:
		err = fmt.Errorf("unsupported output extension %q", ext)
	}
	if err != nil {
		return err
	}
	return w.Close()
}
