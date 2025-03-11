package main

import (
	"fmt"
	"github.com/liondadev/quick-image-server/server"
	"io"
	"os"
)

func main() {
	s := server.Server{}

	f, err := os.Open("fart.jpg")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	c, err := s.MakeThumbnail("image/jpeg", f)
	if err != nil {
		panic(err)
	}

	ff, err := os.Create("gass.png")
	if err != nil {
		panic(err)
	}

	fmt.Println(io.Copy(ff, c))
}
