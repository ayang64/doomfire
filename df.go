package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

type Inferno struct {
	width   int
	height  int
	grid    []int8
	buffer  *bytes.Buffer
	renders int
}

type Dimensions struct {
	Width  int
	Height int
}

func MapColor(v int8) (uint8, uint8, uint8) {
	cmap := [][3]uint8{
		{0x07, 0x07, 0x07}, {0x1f, 0x07, 0x07}, {0x2f, 0x0f, 0x07}, {0x47, 0x0f, 0x07}, {0x57, 0x17, 0x07},
		{0x67, 0x1f, 0x07}, {0x77, 0x1f, 0x07}, {0x8f, 0x27, 0x07}, {0x9f, 0x2f, 0x07}, {0xaf, 0x3f, 0x07},
		{0xbf, 0x47, 0x07}, {0xc7, 0x47, 0x07}, {0xdf, 0x4f, 0x07}, {0xdf, 0x57, 0x07}, {0xdf, 0x57, 0x07},
		{0xd7, 0x5f, 0x07}, {0xd7, 0x67, 0x0f}, {0xcf, 0x6f, 0x0f}, {0xcf, 0x77, 0x0f}, {0xcf, 0x7f, 0x0f},
		{0xcf, 0x87, 0x17}, {0xc7, 0x87, 0x17}, {0xc7, 0x8f, 0x17}, {0xc7, 0x97, 0x1f}, {0xbf, 0x9f, 0x1f},
		{0xbf, 0x9f, 0x1f}, {0xbf, 0xa7, 0x27}, {0xbf, 0xa7, 0x27}, {0xbf, 0xaf, 0x2f}, {0xb7, 0xaf, 0x2f},
		{0xb7, 0xb7, 0x2f}, {0xb7, 0xb7, 0x37}, {0xcf, 0xcf, 0x6f}, {0xdf, 0xdf, 0x9f}, {0xef, 0xef, 0xc7},
		{0xff, 0xff, 0xff},
	}

	if v < 0 || int(v) >= len(cmap) {
		return 0, 0, 0
	}

	return cmap[v][0], cmap[v][1], cmap[v][2]
}

func (i *Inferno) SetDimensions(d Dimensions) {
	i.width = d.Width
	i.height = d.Height
}

func (i *Inferno) Init() {
	// initialize our fire grid
	i.grid = make([]int8, i.width*i.height)
	for j := 0; j < i.width; j++ {
		i.grid[((i.height-1)*i.width)+j] = 35
	}
}

func (i *Inferno) Spread() {
	for y := i.height - 1; y > 0; y-- {
		for x := 0; x < i.width; x++ {

			src := (y * i.width) + x

			// generate random number between [0, 6) and and subtract 3 from it.
			// this biases the results to < 0 which shifts the direction of the
			// flames to the left giving a wind effect.
			dst := (src - i.width) + rand.Intn(6) - 3

			// if destination is outside of the bounds of our display, skip it.
			if start, end := (y-1)*i.width, y*i.width+i.width; dst < start || dst > end {
				continue
			}

			if end := (i.width * i.height) - 1; dst > end {
				dst = end
			}

			// sometimes the flames get a little more intense as they rise.
			i.grid[dst] = i.grid[src] - int8(rand.Intn(7)-1)

			if i.grid[dst] > 35 {
				i.grid[dst] = 35
			}

			if i.grid[dst] < 0 {
				i.grid[dst] = 0
			}
		}
	}
}

func (i *Inferno) Render() {
	i.buffer.WriteString("\x1b[0;0H")

	for y := 0; y < i.height; y += 2 {
		for x := 0; x < i.width; x++ {

			// if necessary, change foreground color
			if pos := (y * i.width) + x; true {
				r, g, b := MapColor(i.grid[pos])
				i.buffer.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b))
			}

			// if necessary, change background color
			if pos := ((y + 1) * i.width) + x; true {
				r, g, b := MapColor(i.grid[pos])
				i.buffer.WriteString(fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b))
			}

			i.buffer.WriteString("▀")
		}
	}

	i.renders++

	io.Copy(os.Stdout, i.buffer)
	i.buffer.Reset()
	time.Sleep(100 * time.Millisecond)
}

func WithDimentions(width int, height int) func(*Inferno) error {
	return func(i *Inferno) error {
		i.width = width
		i.height = height
		return nil
	}
}

func NewInferno(opts ...func(*Inferno) error) (*Inferno, error) {
	rc := Inferno{}

	rc.buffer = &bytes.Buffer{}

	for _, opt := range opts {
		if err := opt(&rc); err != nil {
			return nil, err
		}
	}
	return &rc, nil
}

func fire(ctx context.Context, width int, height int) chan Dimensions {
	rc := make(chan Dimensions)

	go func() {
		inferno, err := NewInferno(WithDimentions(width, height))

		if err != nil {
			return
		}

		inferno.Init()

		for {
			select {
			case <-ctx.Done():
				rc <- Dimensions{}
				return

			case dims := <-rc:
				inferno.SetDimensions(dims)
				inferno.Init()

			default:
				// display grid
				inferno.Render()

				// percollate values up
				inferno.Spread()
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

	dims <- Dimensions{Width: width, Height: height * 2}

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
				dims <- Dimensions{Width: width, Height: height * 2}
			case syscall.SIGINT:
				cancel()
			}
		}
	}

	os.Stdout.Write([]byte("\x1b[39;m"))
	os.Stdout.Write([]byte("\x1b[49;m"))
}
