package docker

import (
    "github.com/fsouza/go-dockerclient"
    "fmt"
    "log"
    "os"
)

const DOCKER_STOP_TIMEOUT = 10 // seconds

type Options struct {
    DockerEndpoint  string              `long:"docker-endpoint"`

    Logger          *log.Logger
}

type Manager struct {
    log             *log.Logger

    dockerClient    *docker.Client
    dockerName      string
}

func NewManager(options Options) (*Manager, error) {
    if options.Logger == nil {
        options.Logger = log.New(os.Stderr, "docker.Manager: ", log.LstdFlags)
    }

    manager := &Manager{
        log:    options.Logger,
    }

    if options.DockerEndpoint == "" {
        if dockerClient, err := docker.NewClientFromEnv(); err != nil {
            return nil, fmt.Errorf("docker.NewClientFromEnv: %v", err)
        } else {
            manager.dockerClient = dockerClient
        }
    } else {
        if dockerClient, err := docker.NewClient(options.DockerEndpoint); err != nil {
            return nil, fmt.Errorf("docker.NewClient: %v", err)
        } else {
            manager.dockerClient = dockerClient
        }
    }

    if dockerInfo, err := manager.dockerClient.Info(); err != nil {
        return nil, fmt.Errorf("dockerClient.Info: %v", err)
    } else {
        manager.dockerName = dockerInfo.Get("name")
    }

    return manager, nil
}

func (manager *Manager) List() (containers []*Container, err error) {
    opts := docker.ListContainersOptions{
        All:        true,
        Filters:    map[string][]string{
            "label":    []string{"close"},
        },
    }

    if listContainers, err := manager.dockerClient.ListContainers(opts); err != nil {
        return nil, err
    } else {
        for _, listContainer := range listContainers {
            if container, err := manager.Get(listContainer.ID); err != nil {
                return nil, err
            } else {
                containers = append(containers, container)
            }
        }
    }

    return containers, nil
}

func (manager *Manager) Get(id string) (*Container, error) {
    dockerContainer, err := manager.dockerClient.InspectContainer(id)
    if _, ok := err.(*docker.NoSuchContainer); ok {
        return nil, nil

    } else if err != nil {
        return nil, fmt.Errorf("dockerClient.InspectContainer %v: %v", id, err)
    }

    container := Container{
        Config: configFromDocker(dockerContainer),
    }

    // ID
    if err := container.parseID(dockerContainer.Name, dockerContainer.Config.Labels); err != nil {
        return nil, fmt.Errorf("parseID %s: %v", dockerContainer.Name, err)
    }

    container.updateStatus(dockerContainer)

    return &container, nil
}

func (manager *Manager) Up(id ID, config Config) (*Container, error) {
    // check
    container, err := manager.Get(id.String())

    if err != nil {
        return container, err
    } else if container == nil {
        // create

    } else if config.Equals(container.Config) {
        manager.log.Printf("Up %v: exists\n", container)

    } else {
        manager.log.Printf("Up %v: old-config %#v\n", container, container.Config)
        manager.log.Printf("Up %v: new-config %#v\n", container, config)

        // cleanup to replace with our config
        manager.log.Printf("Up %v: destroy %v...\n", container, container.ID)

        if err := manager.dockerClient.RemoveContainer(docker.RemoveContainerOptions{ID: container.ContainerID, Force: true}); err != nil {
            return container, fmt.Errorf("dockerClient.RemoveContainer %v: %v", container.ID, err)
        }

        // create
        container = nil
    }

    if container == nil {
        container = &Container{ID:id, Config: config}

        // does not exist; create
        createOptions := docker.CreateContainerOptions{
            Name:   container.String(),
            Config: &docker.Config{
                Env:        config.Env,
                Cmd:        config.Argv(),
                Image:      config.Image,
                // Mounts:     config.Mounts,
                Labels:     id.labels(),
            },
            HostConfig: &docker.HostConfig{
                Privileged:     config.Privileged,
                NetworkMode:    config.NetworkMode,
            },
        }

        if config.NetworkMode == "" {
            // match hostname from container name, unless running with NetworkMode=container:*
            createOptions.Config.Hostname = container.String()
        }

        // XXX: .Config.Mounts = ... doesn't work? fake it!
        createOptions.Config.Volumes = make(map[string]struct{})
        for _, mount := range config.Mounts {
            createOptions.Config.Volumes[mount.Destination] = struct{}{}

            if mount.Source != "" {
                bind := fmt.Sprintf("%s:%s:%s", mount.Source, mount.Destination, mount.Mode)

                createOptions.HostConfig.Binds = append(createOptions.HostConfig.Binds, bind)
            }
        }

        manager.log.Printf("Up %v: create...\n", container)

        if dockerContainer, err := manager.dockerClient.CreateContainer(createOptions); err != nil {
            return nil, err
        } else {
            // status
            container.updateStatus(dockerContainer)
        }
    }

    // running
    if container.IsUp() {
        manager.log.Printf("Up %v: running\n", container)
    } else if err := manager.dockerClient.StartContainer(container.ContainerID, nil); err != nil {
        return nil, fmt.Errorf("dockerClient.StartContainer %v: %v", container.ContainerID, err)
    } else {
        manager.log.Printf("Up %v: started\n", container)

        container.State.Running = true // XXX
    }

    return container, nil
}

func (manager *Manager) Down(container *Container) error {
    manager.log.Printf("Down %v: stopping..\n", container)

    if err := manager.dockerClient.StopContainer(container.ContainerID, DOCKER_STOP_TIMEOUT); err == nil {

    } else if err, isNotRunning := err.(*docker.ContainerNotRunning); isNotRunning {
        // skip
    } else {
        return err
    }

    container.State.Running = false // XXX

    return nil
}

// Stop all and all running containers
func (manager *Manager) Panic() (retErr error) {
    opts := docker.ListContainersOptions{
        Filters:    map[string][]string{
            "label":    []string{"close"},
        },
    }

    if listContainers, err := manager.dockerClient.ListContainers(opts); err != nil {
        return err
    } else {
        for _, listContainer := range listContainers {
            if err := manager.dockerClient.KillContainer(docker.KillContainerOptions{ID: listContainer.ID, Signal: 9}); err != nil {
                manager.log.Printf("Panic: dockerClient.KillContainer %v: %v\n", listContainer.ID, err)
                retErr = err
            } else {
                manager.log.Printf("Panic %v: killed\n", listContainer.Names[0])
            }
        }
    }

    return retErr
}
