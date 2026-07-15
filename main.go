package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"net/http"
	"os/exec"
	"time"

	"radio/internal/station"
	"radio/internal/web"

	"github.com/getlantern/systray"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	mgr, err := station.NewManager(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	handler := web.NewHandler(mgr, mgr.Port(), mgr.MusicDir())

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	addr := fmt.Sprintf(":%d", mgr.Port())
	webURL := fmt.Sprintf("http://localhost%s", addr)

	server := &http.Server{Addr: addr, Handler: mux}

	// Start HTTP server in background
	go func() {
		log.Printf("Radio server starting on %s", webURL)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Build systray icon
	iconData := makeIcon()

	// systray blocks until Quit is clicked
	systray.Run(func() {
		systray.SetIcon(iconData)
		systray.SetTitle("Radio Server")
		systray.SetTooltip("Radio Server")

		openUI := systray.AddMenuItem("Open Web UI", "Open in browser")
		systray.AddSeparator()

		// Station stream URLs
		for _, st := range mgr.AllStations() {
			streamURL := fmt.Sprintf("%s/stream/%s", webURL, st.Mount())
			item := systray.AddMenuItem(st.Name(), streamURL)
			go func(url string) {
				for range item.ClickedCh {
					exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
				}
			}(streamURL)
		}

		systray.AddSeparator()
		quit := systray.AddMenuItem("Quit", "Stop server")

		go func() {
			<-openUI.ClickedCh
			exec.Command("rundll32", "url.dll,FileProtocolHandler", webURL).Start()
		}()

		go func() {
			<-quit.ClickedCh
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			server.Shutdown(ctx)
			systray.Quit()
		}()
	}, func() {
		// onExit — clean shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	})
}

// makeIcon generates a 16x16 .ico file (green circle on transparent background).
func makeIcon() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	green := color.RGBA{0, 180, 0, 255}
	cx, cy, r := 15, 15, 12
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r*r {
				img.Set(x, y, green)
			}
		}
	}
	return rgbaToICO(img)
}

// rgbaToICO converts an RGBA image to .ico format (single 32bpp image).
func rgbaToICO(img *image.RGBA) []byte {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Build BMP data: BITMAPINFOHEADER + pixels (bottom-up BGRA) + AND mask
	pixels := make([]byte, w*h*4)
	rowSize := w * 4
	for y := 0; y < h; y++ {
		srcY := h - 1 - y // bottom-up
		src := img.Pix[srcY*img.Stride : srcY*img.Stride+w*4]
		dst := pixels[y*rowSize : (y+1)*rowSize]
		// RGBA -> BGRA
		for x := 0; x < w; x++ {
			dst[x*4+0] = src[x*4+2] // B
			dst[x*4+1] = src[x*4+1] // G
			dst[x*4+2] = src[x*4+0] // R
			dst[x*4+3] = src[x*4+3] // A
		}
	}

	// AND mask: 1 bit per pixel, rows padded to 4 bytes
	andRowBytes := (w + 31) / 32 * 4
	andMask := make([]byte, andRowBytes*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := img.Pix[(h-1-y)*img.Stride+x*4+3]
			if a < 128 { // transparent -> 1 in AND mask
				andMask[y*andRowBytes+x/8] |= 1 << (7 - uint(x%8))
			}
		}
	}

	bmpSize := 40 + len(pixels) + len(andMask)

	var buf bytes.Buffer

	// ICO header
	buf.Write([]byte{0, 0, 1, 0, 1, 0}) // reserved=0, type=1(ICO), count=1

	// Directory entry: width, height (0 = 256), colors, reserved, planes, bpp, size, offset
	ww, hh := byte(w), byte(h)
	if w == 256 {
		ww = 0
	}
	if h == 256 {
		hh = 0
	}
	buf.Write([]byte{ww, hh, 0, 0, 1, 0, 32, 0}) // width, height, 0 colors, reserved, 1 plane, 32bpp
	write32le(&buf, uint32(bmpSize))                 // size
	write32le(&buf, 22)                              // offset (6 header + 16 entry = 22)

	// BITMAPINFOHEADER
	write32le(&buf, 40)            // header size
	write32le(&buf, uint32(w))     // width
	write32le(&buf, uint32(h*2))   // height (doubled: image + AND mask)
	buf.Write([]byte{1, 0})        // planes
	buf.Write([]byte{32, 0})       // bpp
	write32le(&buf, 0)             // compression (BI_RGB)
	write32le(&buf, 0)             // image size (can be 0 for BI_RGB)
	write32le(&buf, 0)             // x pixels per meter
	write32le(&buf, 0)             // y pixels per meter
	write32le(&buf, 0)             // colors used
	write32le(&buf, 0)             // important colors

	buf.Write(pixels)
	buf.Write(andMask)

	return buf.Bytes()
}

func write32le(buf *bytes.Buffer, v uint32) {
	buf.Write([]byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)})
}
