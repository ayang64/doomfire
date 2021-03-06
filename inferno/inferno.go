package inferno

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"
)

type Flame struct {
	width   int
	height  int
	grid    []int8
	buffer  *bytes.Buffer
	renders int
	rand    *rand.Rand
}

type Dimensions struct {
	Width  int
	Height int
}

func MapColor(v int8) [3]uint8 {
	cmap := [][3]uint8{
		{0x07, 0x07, 0x07}, {0x1f, 0x07, 0x07}, {0x2f, 0x0f, 0x07},
		{0x47, 0x0f, 0x07}, {0x57, 0x17, 0x07}, {0x67, 0x1f, 0x07},
		{0x77, 0x1f, 0x07}, {0x8f, 0x27, 0x07}, {0x9f, 0x2f, 0x07},
		{0xaf, 0x3f, 0x07}, {0xbf, 0x47, 0x07}, {0xc7, 0x47, 0x07},
		{0xdf, 0x4f, 0x07}, {0xdf, 0x57, 0x07}, {0xdf, 0x57, 0x07},
		{0xd7, 0x5f, 0x07}, {0xd7, 0x67, 0x0f}, {0xcf, 0x6f, 0x0f},
		{0xcf, 0x77, 0x0f}, {0xcf, 0x7f, 0x0f}, {0xcf, 0x87, 0x17},
		{0xc7, 0x87, 0x17}, {0xc7, 0x8f, 0x17}, {0xc7, 0x97, 0x1f},
		{0xbf, 0x9f, 0x1f}, {0xbf, 0x9f, 0x1f}, {0xbf, 0xa7, 0x27},
		{0xbf, 0xa7, 0x27}, {0xbf, 0xaf, 0x2f}, {0xb7, 0xaf, 0x2f},
		{0xb7, 0xb7, 0x2f}, {0xb7, 0xb7, 0x37}, {0xcf, 0xcf, 0x6f},
		{0xdf, 0xdf, 0x9f}, {0xef, 0xef, 0xc7}, {0xff, 0xff, 0xff},
	}

	if v < 0 || int(v) >= len(cmap) {
		return [3]uint8{0, 0, 0}
	}

	return cmap[v]
}

func (i *Flame) SetDimensions(d Dimensions) {
	i.width = d.Width
	i.height = d.Height
	i.Init()
}

func (i *Flame) Init() {
	// initialize our fire grid

	// the bottom most row is the source of our "flame" and is set to the higest
	// possible value on the grid.
	//
	// this value is "spread" upward.
	i.grid = make([]int8, i.width*i.height)
	for j := 0; j < i.width; j++ {
		i.grid[((i.height-1)*i.width)+j] = 35
	}
}

func (i *Flame) Spread() {
	for y := i.height - 1; y > 0; y-- {
		for x := 0; x < i.width; x++ {
			src := (y * i.width) + x

			// generate random number between [0, 6) and and subtract 3 from it.
			// this biases the results to < 0 which shifts the direction of the
			// flames to the left giving a wind effect.
			dst := (src - i.width) + i.rand.Intn(6) - 2

			// if destination is outside of the bounds of our display, skip it.
			if start, end := (y-1)*i.width, y*i.width+i.width; dst < start || dst > end {
				continue
			}

			if end := (i.width * i.height) - 1; dst > end {
				dst = end
			}

			// sometimes the flames get a little more intense as they rise.
			i.grid[dst] = i.grid[src] - int8(i.rand.Intn(6)-1)

			// clip grid values to within our range.
			if i.grid[dst] > 35 {
				i.grid[dst] = 35
			}

			if i.grid[dst] < 0 {
				i.grid[dst] = 0
			}
		}
	}
}

func (i *Flame) Render() {
	i.buffer.WriteString("\x1b[0;0H")

	prevbg, prevfg := [3]uint8{}, [3]uint8{}
	for y := 0; y < i.height; y += 2 {
		for x := 0; x < i.width; x++ {
			// if necessary, change foreground color
			if c := MapColor(i.grid[(y*i.width)+x]); c != prevfg {
				i.buffer.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm", c[0], c[1], c[2]))
				prevfg = c
			}
			// if necessary, change background color
			if c := MapColor(i.grid[((y+1)*i.width)+x]); c != prevbg {
				i.buffer.WriteString(fmt.Sprintf("\x1b[48;2;%d;%d;%dm", c[0], c[1], c[2]))
				prevbg = c
			}
			i.buffer.WriteString("▀")
		}
	}

	i.renders++

	io.Copy(os.Stdout, i.buffer)
	i.buffer.Reset()
	time.Sleep(100 * time.Millisecond)
}

func WithDimentions(width int, height int) func(*Flame) error {
	return func(i *Flame) error {
		i.width = width
		i.height = height
		return nil
	}
}

func NewFlame(opts ...func(*Flame) error) (*Flame, error) {
	rc := Flame{}

	rc.buffer = &bytes.Buffer{}
	rc.rand = rand.New(rand.NewSource(time.Now().UnixNano()))

	for _, opt := range opts {
		if err := opt(&rc); err != nil {
			return nil, err
		}
	}
	return &rc, nil
}
