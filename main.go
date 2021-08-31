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
			Name:   "host",
			Usage:  "docker host endpoint",
			Value:  "unix://var/socket/docker.sock",
			EnvVar: "PLUGIN_HOST",
		},
		cli.BoolFlag{
			Name:   "dry_run",
			Usage:  "dry run disables docker push",
			EnvVar: "PLUGIN_DRY_RUN",
		},
		cli.StringFlag{
			Name:   "docker_api_version",
			Usage:  "specify version of docker api",
			Value:  "1.40",
			EnvVar: "PLUGIN_DOCKER_API_VERSION",
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
			Name:   "tags",
			Usage:  "list of image tags",
			Value:  &cli.StringSlice{},
			EnvVar: "PLUGIN_TAGS",
		},
		cli.StringFlag{
			Name:   "tagfile",
			Usage:  "specify tag file",
			Value:  ".tags",
			EnvVar: "PLUGIN_TAGFILE",
		},
		cli.BoolFlag{
			Name:   "tag_latest",
			Usage:  "tag latest version",
			EnvVar: "PLUGIN_TAG_LATEST",
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
			TagFile:    c.String("tagfile"),
			TagLatest:  c.Bool("tag_latest"),
			DryRun:     c.Bool("dry_run"),
			Dockerfile: absDocker,
			Version:    c.String("docker_api_version"),
			Registries: registries,
			Images:     images,
			Tags:       c.StringSlice("tags"),
		},
	}
	return plugin.Exec()
}
