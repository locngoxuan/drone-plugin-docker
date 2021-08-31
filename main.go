package main

import (
	"encoding/json"
	"io/ioutil"
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
		cli.StringSliceFlag{
			Name:   "registry_envs",
			Usage:  "list of environment registry variables",
			Value:  &cli.StringSlice{},
			EnvVar: "PLUGIN_REGISTRY_ENVS",
		},
		cli.StringFlag{
			Name:   "registry",
			Usage:  "specify registry configuration file",
			Value:  "",
			EnvVar: "PLUGIN_REGISTRY",
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
	registryEnvs := c.StringSlice("registry_envs")

	registries := make(map[string]DockerRegistry)
	for _, key := range registryEnvs {
		// read from settings
		v := strings.TrimSpace(os.Getenv(key))
		var elem DockerAuth
		err = json.Unmarshal([]byte(v), &elem)
		if err != nil {
			return err
		}
		for registry, auth := range elem.Auths {
			if v := strings.TrimSpace(auth.Auth); v != "" {
				auth.Username, auth.Password = decode(v)
			}
			log.Printf("add registry %s", registry)
			registries[registry] = auth
		}
	}

	v := strings.TrimSpace(c.String("registry"))
	if v != "" {
		// read from registry file
		bs, err := ioutil.ReadFile(v)
		if err != nil {
			return err
		}
		var elem DockerAuth
		err = json.Unmarshal(bs, &elem)
		if err != nil {
			return err
		}
		for registry, auth := range elem.Auths {
			if v := strings.TrimSpace(auth.Auth); v != "" {
				auth.Username, auth.Password = decode(v)
			}
			log.Printf("add registry %s", registry)
			registries[registry] = auth
		}
	}

	plugin := &Plugin{
		Config: Config{
			Src:        pwd,
			TagFile:    c.String("tagfile"),
			TagLatest:  c.Bool("tag_latest"),
			Dockerfile: absDocker,
			Version:    c.String("docker_api_version"),
			Registries: registries,
			Images:     c.StringSlice("images"),
			Tags:       c.StringSlice("tags"),
		},
	}
	return plugin.Exec()
}
