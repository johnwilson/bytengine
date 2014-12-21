package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/codegangsta/cli"
)

var sh *Shell

func main() {
	// handle SIGINT
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt)

	go func() {
		for _ = range done {
			if sh != nil {
				sh.Close()
			}
			fmt.Print("\nbye\n")
			os.Exit(0)
		}
	}()

	app := cli.NewApp()
	run := cli.Command{
		Name: "run",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "u", Value: "", Usage: "username"},
			cli.StringFlag{Name: "p", Value: "", Usage: "password"},
			cli.StringFlag{Name: "host", Value: "localhost"},
			cli.IntFlag{Name: "port", Value: 8500},
			cli.StringFlag{Name: "editor", Value: "vim"},
		},
		Action: func(c *cli.Context) {
			sh = NewShell()
			defer sh.Close()
			// get options
			username := c.String("u")
			password := c.String("p")
			host := c.String("host")
			port := c.Int("port")
			editor := c.String("editor")

			// connect
			err := sh.BEClient.Connect(host, port)
			if err != nil {
				log.Fatal(err)
			}
			// login
			err = sh.BEClient.Login(username, password)
			if err != nil {
				log.Fatal(err)
			}

			// start shell
			if err = sh.Init(editor); err != nil {
				log.Fatal(err)
			}
			sh.Start()
		},
	}
	app.Commands = []cli.Command{run}
	app.Run(os.Args)
}
