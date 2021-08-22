package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/urfave/cli"
)

var (
	version = "unknown"
)

func main() {
	// Load env-file if it exists first
	if env := os.Getenv("PLUGIN_ENV_FILE"); env != "" {
		godotenv.Load(env)
	}

	app := cli.NewApp()
	app.Name = "docker plugin"
	app.Usage = "docker plugin"
	app.Action = run
	app.Version = version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "remote.url",
			Usage:  "git remote url",
			EnvVar: "DRONE_REMOTE_URL",
		},
		cli.StringFlag{
			Name:   "commit.sha",
			Usage:  "git commit sha",
			EnvVar: "DRONE_COMMIT_SHA",
			Value:  "00000000",
		},
		cli.StringFlag{
			Name:   "commit.ref",
			Usage:  "git commit ref",
			EnvVar: "DRONE_COMMIT_REF",
		},
		cli.StringFlag{
			Name:   "host",
			Usage:  "docker host endpoint",
			Value:  "unix://var/socket/docker.sock",
			EnvVar: "PLUGIN_HOST",
		},
		cli.BoolFlag{
			Name:   "dry-run",
			Usage:  "dry run disables docker push",
			EnvVar: "PLUGIN_DRY_RUN",
		},
		cli.StringFlag{
			Name:   "dockerfile",
			Usage:  "build dockerfile",
			Value:  "Dockerfile",
			EnvVar: "PLUGIN_DOCKERFILE",
		},
		cli.StringFlag{
			Name:   "context",
			Usage:  "build context",
			Value:  ".",
			EnvVar: "PLUGIN_CONTEXT",
		},
		cli.StringFlag{
			Name:   "registries",
			Usage:  "list of private docker registries",
			Value:  "",
			EnvVar: "PLUGIN_REGISTRIES",
		},
		cli.StringSliceFlag{
			Name:   "images",
			Usage:  "list of output images",
			Value:  &cli.StringSlice{},
			EnvVar: "PLUGIN_IMAGES",
		},
		cli.StringSliceFlag{
			Name:   "versions",
			Usage:  "list of image version",
			Value:  &cli.StringSlice{"latest"},
			EnvVar: "PLUGIN_VERSIONS",
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	pwd, err := filepath.Abs(c.String("context"))
	if err != nil {
		return err
	}
	log.Printf("docker build context: %s", pwd)
	absDocker, err := filepath.Abs(c.String("dockerfile"))
	if err != nil {
		return err
	}
	log.Printf("dockerfile: %s", absDocker)
	registrieJson := c.String("registries")
	var registries []DockerRegistry
	if strings.TrimSpace(registrieJson) != "" {
		err = json.Unmarshal([]byte(registrieJson), &registries)
		if err != nil {
			return err
		}
	}

	imageJson := c.String("images")
	var images []OutputDockerImage
	if strings.TrimSpace(imageJson) != "" {
		err = json.Unmarshal([]byte(imageJson), &images)
		if err != nil {
			return err
		}
	}

	plugin := &Plugin{
		Config: Config{
			Src:        pwd,
			DryRun:     c.Bool("dry-run"),
			Dockerfile: absDocker,
			Registries: registries,
			Images:     images,
			Versions:   c.StringSlice("versions"),
		},
	}
	return plugin.Exec()
}
