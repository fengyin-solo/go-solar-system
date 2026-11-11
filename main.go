package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/furan917/go-solar-system/internal/app"
)

// cliOptions holds the parsed command line configuration.
type cliOptions struct {
	offline bool
}

// parseCLIArgs parses command line arguments. It is split out of main so the
// CLI contract can be tested without starting the terminal application.
func parseCLIArgs(args []string) (cliOptions, error) {
	flags := flag.NewFlagSet("go-solar-system", flag.ContinueOnError)
	offline := flags.Bool(
		"offline",
		false,
		"skip the remote API and use the built-in offline solar system definition",
	)

	if err := flags.Parse(args); err != nil {
		return cliOptions{}, err
	}

	return cliOptions{offline: *offline}, nil
}

func main() {
	options, err := parseCLIArgs(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		log.Fatal(err)
	}

	bundled, err := fs.Sub(bundledSystems, "systems")
	if err != nil {
		log.Fatal(err)
	}

	var appOpts []app.Option
	if options.offline {
		appOpts = append(appOpts, app.WithOffline())
	}

	solarSystem, err := app.NewSolarSystem(bundled, appOpts...)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("🌌 Welcome to the Interactive Solar System!")
	if err := solarSystem.Run(); err != nil {
		log.Fatal(err)
	}
}
