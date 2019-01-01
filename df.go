package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ayang64/doomfire/inferno"

	"golang.org/x/crypto/ssh/terminal"
)

func fire(ctx context.Context, width int, height int) chan inferno.Dimensions {
	rc := make(chan inferno.Dimensions)

	go func() {
		inf, err := inferno.NewFlame(inferno.WithDimentions(width, height))

		if err != nil {
			return
		}

		inf.Init()

		for {
			select {
			case <-ctx.Done():
				rc <- inferno.Dimensions{}
				return

			case dims := <-rc:
				inf.SetDimensions(dims)
				inf.Init()

			default:
				// display grid
				inf.Render()

				// percollate values up
				inf.Spread()
			}
		}
	}()

	return rc
}

func main() {
	rand.Seed(time.Now().UnixNano())

	width, height, err := terminal.GetSize(0)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	dims := fire(ctx, width, height*2)

	dims <- inferno.Dimensions{Width: width, Height: height * 2}

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGWINCH, syscall.SIGINT)

mainloop:
	for {
		select {
		case <-dims:
			// received signal indicating flame goroutine has ended.
			// we can safely exit now.
			break mainloop

		case sig := <-sigs:
			switch sig {
			case syscall.SIGWINCH:
				width, height, _ := terminal.GetSize(0)
				dims <- inferno.Dimensions{Width: width, Height: height * 2}
			case syscall.SIGINT:
				cancel()
			}
		}
	}

	os.Stdout.Write([]byte("\x1b[39;m"))
	os.Stdout.Write([]byte("\x1b[49;m"))
}
