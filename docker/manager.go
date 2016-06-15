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
        manager.dockerName = dockerInfo.Name
    }

    return manager, nil
}

// Get short container status for given class
func (manager *Manager) List(filter ID) (containers []ContainerStatus, err error) {
    labelFilter := []string{}

    if filter.Class == "" {
        labelFilter = append(labelFilter, "close")
    } else {
        labelFilter = append(labelFilter, fmt.Sprintf("close=%s", filter.Class))
    }

    if filter.Type != "" {
        labelFilter = append(labelFilter, fmt.Sprintf("close.type=%s", filter.Type))
    }
    if filter.Instance != "" {
        labelFilter = append(labelFilter, fmt.Sprintf("close.instance=%s", filter.Instance))
    }

    opts := docker.ListContainersOptions{
        All:        true,
        Filters:    map[string][]string{
            "label":    labelFilter,
        },
    }

    if listContainers, err := manager.dockerClient.ListContainers(opts); err != nil {
        return nil, err
    } else {
        for _, listContainer := range listContainers {
            container := ContainerStatus{}

            // ID
            if err := container.parseID(listContainer.Names[0], listContainer.Labels); err != nil {
                return nil, fmt.Errorf("parseID %s: %v", listContainer.Names, err)
            }

            // Status + Config
            if err := container.fromDockerList(listContainer); err != nil {
                return nil, err
            }


            containers = append(containers, container)
        }
    }

    return containers, nil
}

// Get complete container state for given container
func (manager *Manager) Get(id string) (*Container, error) {
    dockerContainer, err := manager.dockerClient.InspectContainer(id)
    if _, ok := err.(*docker.NoSuchContainer); ok {
        return nil, nil

    } else if err != nil {
        return nil, fmt.Errorf("dockerClient.InspectContainer %v: %v", id, err)
    }

    container := Container{}

    // ID
    if err := container.parseID(dockerContainer.Name, dockerContainer.Config.Labels); err != nil {
        return nil, fmt.Errorf("parseID %s: %v", dockerContainer.Name, err)
    }

    // Status + Config
    if err := container.update(dockerContainer); err != nil {
        return nil, err
    }

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
        manager.log.Printf("Up %v: exists\n", id)

    } else {
        manager.log.Printf("Up %v: old-config %#v\n", id, container.Config)
        manager.log.Printf("Up %v: new-config %#v\n", id, config)

        // cleanup to replace with our config
        manager.log.Printf("Up %v: destroy %v...\n", id, container.ID)

        if err := manager.dockerClient.RemoveContainer(docker.RemoveContainerOptions{ID: container.ContainerID, Force: true}); err != nil {
            return container, fmt.Errorf("dockerClient.RemoveContainer %v: %v", container.ID, err)
        }

        // create
        container = nil
    }

    if container == nil {
        // does not exist; create
        manager.log.Printf("Up %v: create...\n", id)

        createOptions := config.createOptions(id)

        if dockerContainer, err := manager.dockerClient.CreateContainer(createOptions); err != nil {
            return nil, err
        } else {
            // XXX: the response is not actually a full docker.Container...
            container = &Container{
                ContainerStatus: ContainerStatus{
                    ID:             id,
                    ContainerID:    dockerContainer.ID,
                },
                Config:         config,
            }
        }
    }

    // running
    if container.IsUp() {
        manager.log.Printf("Up %v: running\n", container)
    } else if err := manager.dockerClient.StartContainer(container.ContainerID, nil); err != nil {
        return nil, fmt.Errorf("dockerClient.StartContainer %v: %v", container.ContainerID, err)
    } else {
        manager.log.Printf("Up %v: started\n", container)

        // XXX: should watch containers and get their state from there..
        container.State = "running"
        container.ContainerState.Running = true
    }

    return container, nil
}

// Stop docker container, if running. Ignored if already stopped.
//
// Leaves the container in place, ready to be restarted
func (manager *Manager) Down(id ID) error {
    manager.log.Printf("Down %v: stopping..\n", id)

    if err := manager.dockerClient.StopContainer(id.String(), DOCKER_STOP_TIMEOUT); err == nil {

    } else if err, isNotRunning := err.(*docker.ContainerNotRunning); isNotRunning {
        // skip
    } else {
        return err
    }

    return nil
}

// Force-kill all running managed containers
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

// Clean up a docker container, removing it.
//
// Force-stops running containers.
func (manager *Manager) Clean(id ID) error {
    manager.log.Printf("Clean %v: removing..\n", id)

    if err := manager.dockerClient.RemoveContainer(docker.RemoveContainerOptions{ID: id.String(), Force: true}); err != nil {
        return err
    }

    return nil
}
