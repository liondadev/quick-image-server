package main

import (
	"github.com/jmoiron/sqlx"
	"github.com/liondadev/quick-image-server/config"
	"github.com/liondadev/quick-image-server/server"
	"log"
	"os"

	_ "github.com/glebarez/go-sqlite"
)

func main() {
	// Open & Load Config
	f, err := os.Open("config.json")
	if err != nil {
		log.Panicf("open config file: %s", err.Error())
		return
	}
	defer f.Close()

	cfg, err := config.FromReader(f)
	if err != nil {
		log.Panicf("parse config: %s", err.Error())
		return
	}

	// Sqlite connection
	path := cfg.DatabasePath
	if path == "" {
		log.Fatalln("Config didn't provide a 'sqlite' option as a path to an sqlite file.")
		return
	}

	db, err := sqlx.Open("sqlite", path)
	if err != nil {
		log.Fatalf("Failed to open sqlite driver: %s", err.Error())
		return
	}
	defer db.Close()

	svr := server.New(cfg, db)
	err = svr.SetupHTTP()
	if err != nil {
		log.Panicf("setup http: %s", err.Error())
		return
	}

	err = svr.ApplyMigrations()
	if err != nil {
		log.Panicf("Failed to apply database migrations: %s", err.Error())
		return
	}

	log.Panicln(svr.Run(":8080"))
}
