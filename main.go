package main


import (
	"fmt"
	"flag"
	"bytengine/kernel"	
)

func main() {
	// Define flags
	cfile := flag.String("config","/opt/bytengine/conf/config.json","bytengine configuration file path")
	// Parse
	flag.Parse()
	// Start server
	fmt.Println("starting bytengine server ...")
	kernel.Run(*cfile)
}