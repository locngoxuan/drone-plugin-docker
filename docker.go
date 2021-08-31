package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
)

var (
	DefaultDockerUnixSock = "unix:///var/run/docker.sock"
	DefaultDockerTCPSock  = "tcp://127.0.0.1:2375"
)

type DockerClient struct {
	Client *client.Client
}

func (c *DockerClient) Close() {
	if c.Client != nil {
		_ = c.Client.Close()
	}
}

func (c *DockerClient) RemoveImage(ctx context.Context, imageId string) ([]types.ImageDeleteResponseItem, error) {
	return c.Client.ImageRemove(ctx, imageId, types.ImageRemoveOptions{
		PruneChildren: true,
		Force:         true,
	})
}

func (c *DockerClient) BuildImageWithOpts(ctx context.Context, tarFile, dockerFile string, tags []string, reg map[string]DockerRegistry) (types.ImageBuildResponse, error) {
	authConfigs := make(map[string]types.AuthConfig)
	for key, registry := range reg {
		authConfigs[key] = types.AuthConfig{
			Username: registry.Username,
			Password: registry.Password,
		}
	}
	opt := types.ImageBuildOptions{
		NoCache:     true,
		Remove:      true,
		ForceRemove: true,
		Tags:        tags,
		PullParent:  true,
		Dockerfile:  dockerFile,
		AuthConfigs: authConfigs,
	}

	dockerBuildContext, err := os.Open(tarFile)
	if err != nil {
		return types.ImageBuildResponse{}, err
	}

	return c.Client.ImageBuild(ctx, dockerBuildContext, opt)
}

func (c *DockerClient) PushImage(ctx context.Context, username, password, image string) (io.ReadCloser, error) {
	a, err := auth(username, password)
	if err != nil {
		return nil, err
	}
	opt := types.ImagePushOptions{
		RegistryAuth: a,
		All:          true,
	}
	return c.Client.ImagePush(ctx, image, opt)
}

func auth(username, password string) (string, error) {
	authConfig := types.AuthConfig{
		Username: username,
		Password: password,
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", fmt.Errorf("can not read docker log %v", err)
	}
	return base64.URLEncoding.EncodeToString(encodedJSON), nil
}

func connectDockerHost(ctx context.Context, hosts []string, version string) (dockerCli DockerClient, err error) {
	host, err := verifyDockerHostConnection(ctx, hosts, version)
	if err != nil {
		return
	}
	_ = os.Setenv("DOCKER_HOST", host)
	_ = os.Setenv("DOCKER_API_VERSION", version)
	dockerCli.Client, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return
	}
	return
}

func verifyDockerHostConnection(ctx context.Context, dockerHosts []string, version string) (string, error) {
	var err error
	for _, host := range dockerHosts {
		err = os.Setenv("DOCKER_HOST", host)
		if err != nil {
			continue
		}
		err = os.Setenv("DOCKER_API_VERSION", version)
		if err != nil {
			continue
		}
		cli, e := client.NewClientWithOpts(client.FromEnv)
		if e != nil {
			err = e
			continue
		}
		if cli == nil {
			err = fmt.Errorf("can not init docker client")
			continue
		}
		_, err = cli.Info(ctx)
		if err != nil {
			_ = cli.Close()
			continue
		}
		_ = cli.Close()
		return host, nil
	}
	if err != nil {
		return "", fmt.Errorf("connect docker host error: %v", err)
	}
	return "", nil
}

func displayDockerLog(in io.Reader) error {
	var dec = json.NewDecoder(in)
	for {
		var jm jsonmessage.JSONMessage
		if err := dec.Decode(&jm); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if jm.Error != nil {
			return fmt.Errorf(jm.Error.Message)
		}
		if strings.TrimSpace(jm.Stream) == "" {
			continue
		}
		log.Println(strings.TrimSuffix(jm.Stream, "\n"))
	}
	return nil
}

type StreamHandler func(s string)

func StreamDockerLog(in io.Reader, f StreamHandler) {
	r := bufio.NewReader(in)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}
		f(strings.TrimSuffix(line, "\n"))
	}
}

func RemoveAfterDone(cli *client.Client, id string) {
	_ = cli.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{
		Force: true,
	})
}
