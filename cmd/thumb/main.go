package main

import (
	"fmt"
	"io"
	"os"

	"github.com/liondadev/quick-image-server/server"
)

func main() {
	s := server.Server{}

	f, err := os.Open("./thing.jpg")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	c, err := s.MakeBubbleImage("image/jpeg", f)
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
