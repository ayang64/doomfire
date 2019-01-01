package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ayang64/doomfire/inferno"

	"golang.org/x/crypto/ssh/terminal"
)

func fire(ctx context.Context) chan inferno.Dimensions {
	rc := make(chan inferno.Dimensions)

	go func() {
		inf, err := inferno.NewFlame(inferno.WithDimentions(0, 0))

		if err != nil {
			return
		}

		for {
			select {
			case <-ctx.Done():
				rc <- inferno.Dimensions{}
				return

			case dims := <-rc:
				inf.SetDimensions(dims)

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

func run() error {
	width, height, err := terminal.GetSize(0)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	dims := fire(ctx)
	dims <- inferno.Dimensions{Width: width, Height: height * 2}

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGWINCH, syscall.SIGINT)
	go func() {
		for sig := range sigs {
			switch sig {
			case syscall.SIGWINCH:
				width, height, _ := terminal.GetSize(0)
				dims <- inferno.Dimensions{Width: width, Height: height * 2}
			case syscall.SIGINT:
				cancel()
				return
			}
		}
	}()

	<-dims
	os.Stdout.Write([]byte("\x1b[39;m"))
	os.Stdout.Write([]byte("\x1b[49;m"))

	return nil
}

func main() {
	run()
}
