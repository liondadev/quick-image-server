package main

import (
	"fmt"
	"github.com/liondadev/quick-image-server/server"
	"io"
	"os"
)

func main() {
	s := server.Server{}

	f, err := os.Open("kvarkar-exposed.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	c, err := s.MakeBubbleImage("image/png", f)
	if err != nil {
		panic(err)
	}

	ff, err := os.Create("gass.gif")
	if err != nil {
		panic(err)
	}

	gifr, err := s.ImageToGif(c)
	if err != nil {
		panic(err)
	}

	fmt.Println(io.Copy(ff, gifr))
}
