package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jhoonb/archivex"
)

type PushEntity struct {
	Username string
	Password string
	Image    string
	InSecure bool
}

type DockerRegistry struct {
	Address  string `json:"address,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type OutputDockerImage struct {
	Registry string `json:"registry,omitempty"`
	Name     string `json:"name,omitempty"`
}

type Config struct {
	Src        string
	Host       string
	DryRun     bool
	Dockerfile string
	Version    string
	TagFile    string
	TagLatest  bool
	Registries []DockerRegistry
	Images     []OutputDockerImage
	Tags       []string
}

type Plugin struct {
	Config Config
}

func readTagsFromFile(file string) ([]string, error) {
	f, e := os.Open(file)
	if e != nil {
		return nil, e
	}
	defer func() {
		_ = f.Close()
	}()
	var result []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		result = append(result, strings.TrimSpace(line))
	}
	return result, nil
}

func (p Plugin) Exec() error {
	ctx := context.Background()
	hosts := []string{
		DefaultDockerUnixSock, DefaultDockerTCPSock,
	}
	if p.Config.Host == "" {
		hosts = []string{p.Config.Host}
	}
	cli, err := connectDockerHost(ctx, hosts, p.Config.Version)

	if err != nil {
		return err
	}
	defer func() {
		cli.Close()
	}()
	err = os.MkdirAll("/tmp", 0755)
	if err != nil {
		return err
	}
	tarFile, err := p.createDockerBuildContext()
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(tarFile)
	}()

	if len(p.Config.Tags) == 0 {
		tags, err := readTagsFromFile(p.Config.TagFile)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}

		if tags != nil && len(tags) > 0 {
			p.Config.Tags = append(p.Config.Tags, tags...)
		}
	}

	if p.Config.TagLatest {
		hasLatest := false
		for _, v := range p.Config.Tags {
			if v == "latest" {
				hasLatest = true
				break
			}
		}
		if !hasLatest {
			p.Config.Tags = append(p.Config.Tags, "latest")
		}
	}

	var pushes []PushEntity
	var images []string
	for _, image := range p.Config.Images {
		name := image.Name
		if !strings.HasPrefix(name, image.Registry) && strings.TrimSpace(image.Registry) != "" {
			name = strings.TrimSpace(image.Registry) + "/" + image.Name
		}

		uname := ""
		pass := ""
		for _, reg := range p.Config.Registries {
			if reg.Address == image.Registry {
				uname = reg.Username
				pass = reg.Password
			}
		}

		for _, ver := range p.Config.Tags {
			ref := name + ":" + ver
			log.Printf("image: %s", ref)
			images = append(images, ref)
			pushes = append(pushes, PushEntity{
				Username: uname,
				Password: pass,
				Image:    ref,
			})
		}
	}

	if len(images) == 0 {
		return fmt.Errorf("no such image is configured")
	}

	defer func() {
		for _, image := range images {
			_, _ = cli.RemoveImage(context.TODO(), image)
		}
	}()

	_, dockerFileName := filepath.Split(p.Config.Dockerfile)

	response, err := cli.BuildImageWithOpts(ctx, tarFile, dockerFileName, images, p.Config.Registries)
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()
	err = displayDockerLog(response.Body)
	if err != nil {
		return err
	}

	for _, push := range pushes {
		r, err := cli.PushImage(ctx, push.Username, push.Password, push.Image)
		if err != nil {
			return err
		}
		err = displayDockerLog(r)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p Plugin) createDockerBuildContext() (string, error) {
	// tar info
	randomName := time.Now().Format("20060102150405")
	tarFile := filepath.Join("/tmp", fmt.Sprintf("build-%s.tar", randomName))
	//create tar at common directory
	tar := new(archivex.TarFile)
	err := tar.Create(tarFile)
	if err != nil {
		return "", err
	}

	srcDir := p.Config.Src

	fileInfos, err := ioutil.ReadDir(srcDir)
	if err != nil {
		return "", err
	}

	addSingleFile := func(fileInfo os.FileInfo) error {
		fp := filepath.Join(srcDir, fileInfo.Name())
		file, err := os.Open(fp)
		if err != nil {
			return err
		}
		defer func() {
			_ = file.Close()
		}()
		err = tar.Add(fileInfo.Name(), file, fileInfo)
		if err != nil {
			return err
		}
		return nil
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			err = tar.AddAll(filepath.Join(srcDir, fileInfo.Name()), true)
			if err != nil {
				return "", err
			}
		} else {
			err = addSingleFile(fileInfo)
			if err != nil {
				return "", err
			}
		}
	}

	//Add Dockerfile
	dockerFile, err := os.Open(p.Config.Dockerfile)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = dockerFile.Close()
	}()
	dockerFileInfo, _ := dockerFile.Stat()
	err = tar.Add("Dockerfile", dockerFile, dockerFileInfo)
	if err != nil {
		return "", err
	}

	err = tar.Close()
	if err != nil {
		return "", err
	}
	return tarFile, nil
}
