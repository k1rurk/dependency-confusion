package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/api/types/filters"
	log "github.com/sirupsen/logrus"
)

func buildImage(client *client.Client, tags []string, managerName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()

	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)

	tar, err := archive.TarWithOptions(filepath.Join(basepath, managerName, "sources"), &archive.TarOptions{})
	if err != nil {
		return err
	}

	// Define the build options to use for the file
	// https://godoc.org/github.com/docker/docker/api/types#ImageBuildOptions
	buildOptions := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Remove:     true,
		Tags:       tags,
	}

	// Build the actual image
	imageBuildResponse, err := client.ImageBuild(
		ctx,
		tar,
		buildOptions,
	)

	if err != nil {
		return err
	}
	
	// Read the STDOUT from the build process
	defer imageBuildResponse.Body.Close()
	_, err = io.Copy(os.Stdout, imageBuildResponse.Body)
	if err != nil {
		return err
	}

	return nil
}

func createAndRunContainer(client *client.Client, imageName, containerName string) error {
	ctx := context.Background()

	// Creating a container
	resp, err := client.ContainerCreate(ctx, &container.Config{
		Image: imageName,
	},
		nil, nil, nil, containerName,
	)

	if err != nil {
		return err
	}
	// Starting the container
	if err := client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}

	return nil
}

// Stop and remove a container
func stopAndRemoveContainer(client *client.Client, containerName string) error {
	ctx := context.Background()

	if err := client.ContainerStop(ctx, containerName, container.StopOptions{}); err != nil {
		log.Errorf("Unable to stop container %s: %s", containerName, err)
		return err
	}

	removeOptions := container.RemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := client.ContainerRemove(ctx, containerName, removeOptions); err != nil {
		log.Errorf("Unable to remove container: %s", err)
		return err
	}

	return nil
}

func BuildAndRun(managerName, imageName, containerName string) error {
	client, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("unable to create docker client: %s", err)
	}

	// // Get list images
	// images, err := client.ImageList(context.Background(), image.ListOptions{})
	// if err != nil {
	// 	panic(err)
	// }
	
	// founded := false
	// for _, image := range images {
	// 	if image.RepoTags[0] == fmt.Sprintf("%s:latest", imageName) {
	// 		founded = true
	// 	}
	// }

	// if !founded {
	// 	// Client, imagename and Dockerfile location
	// 	tags := []string{imageName}
	// 	err = buildImage(client, tags, managerName)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	// Client, imagename and Dockerfile location
	tags := []string{imageName}
	err = buildImage(client, tags, managerName)
	if err != nil {
		return err
	}
	err = createAndRunContainer(client, imageName, containerName)
	if err != nil {
		return err
	}

	err = stopAndRemoveContainer(client, containerName)
	if err != nil {
		return err
	}

	_, err = client.ImagesPrune(context.Background(), filters.NewArgs())
	if err != nil {
		return err
	}

	return nil
}
