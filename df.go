package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

type Inferno struct {
	width   int
	height  int
	grid    []int8
	Renders int
}

func MapColor(v int8) (uint8, uint8, uint8) {
	cmap := [][3]uint8{{07, 07, 07},
		{0x1f, 0x07, 0x07}, {0x2f, 0x0f, 0x07}, {0x47, 0x0f, 0x07}, {0x57, 0x17, 0x07}, {0x67, 0x1f, 0x07},
		{0x77, 0x1f, 0x07}, {0x8f, 0x27, 0x07}, {0x9f, 0x2f, 0x07}, {0xaf, 0x3f, 0x07}, {0xbf, 0x47, 0x07},
		{0xc7, 0x47, 0x07}, {0xDF, 0x4F, 0x07}, {0xDF, 0x57, 0x07}, {0xDF, 0x57, 0x07}, {0xD7, 0x5F, 0x07},
		{0xD7, 0x67, 0x0F}, {0xcf, 0x6f, 0x0f}, {0xcf, 0x77, 0x0f}, {0xcf, 0x7f, 0x0f}, {0xCF, 0x87, 0x17},
		{0xC7, 0x87, 0x17}, {0xC7, 0x8F, 0x17}, {0xC7, 0x97, 0x1F}, {0xBF, 0x9F, 0x1F}, {0xBF, 0x9F, 0x1F},
		{0xBF, 0xA7, 0x27}, {0xBF, 0xA7, 0x27}, {0xBF, 0xAF, 0x2F}, {0xB7, 0xAF, 0x2F}, {0xB7, 0xB7, 0x2F},
		{0xB7, 0xB7, 0x37}, {0xCF, 0xCF, 0x6F}, {0xDF, 0xDF, 0x9F}, {0xEF, 0xEF, 0xC7}, {0xFF, 0xFF, 0xFF},
	}

	if v < 0 || int(v) >= len(cmap) {
		return 0, 0, 0
	}

	return cmap[v][0], cmap[v][1], cmap[v][2]
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
			dst := (src - i.width) + rand.Intn(4) - 2

			if dst < 0 {
				dst = 0
			}

			i.grid[dst] = i.grid[src] - int8(rand.Intn(6)-1)

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
	rc := bytes.Buffer{}

	// clear screen and send cursor to upper left corner
	rc.Write([]byte("\x1b[48;2;0;0;0m"))
	rc.Write([]byte("\x1b[;f"))

	prev := int8(-1)
	for y := 0; y < i.height; y++ {
		for x := 0; x < i.width; x++ {
			pos := (y * i.width) + x
			// if the color has changed, send apropriate escape sequence
			if i.grid[pos] != prev {
				r, g, b := MapColor(i.grid[pos])
				rc.Write([]byte(fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)))
			}
			rc.WriteString(" ")
			prev = i.grid[pos]
		}
	}

	i.Renders++

	io.Copy(os.Stdout, &rc)
	rc.Reset()
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
	for _, opt := range opts {
		if err := opt(&rc); err != nil {
			return nil, err
		}
	}
	return &rc, nil
}

func fire(width int, height int) error {

	inferno, err := NewInferno(WithDimentions(width, height))

	if err != nil {
		return err
	}

	inferno.Init()

	for {
		// display grid
		inferno.Render()

		// percollate values up
		inferno.Spread()
	}
	return nil
}

func main() {
	width, height, err := terminal.GetSize(0)
	if err != nil {
		log.Fatal(err)
	}
	fire(width, height)
}
