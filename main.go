package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
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

// makeIcon generates a simple 16x16 icon (green circle on transparent background).
func makeIcon() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	green := color.RGBA{0, 180, 0, 255}
	cx, cy, r := 8, 8, 6
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r*r {
				img.Set(x, y, green)
			}
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}
