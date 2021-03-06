package control

// A client is a docker container providing a networking environment for running workers in
// it does not provide any worker config/stats

import (
    "github.com/qmsk/close/docker"
    "fmt"
)

type ClientConfig struct {
    name            string
    Count           uint

    Image           string
    Privileged      bool

    Volume          string
    VolumePath      string
    VolumeFmtID     string
    VolumeReadonly  bool
}

func (self ClientConfig) String() string {
    return self.name
}

type Client struct {
    Config          *ClientConfig
    Instance        string
    up              bool            // configured to be up

    dockerID        docker.ID
}

func (self Client) String() string {
    return fmt.Sprintf("%v:%s", self.Config, self.Instance)
}

func (self *Manager) discoverClient(dockerID docker.ID) (*Client, error) {
    clientConfig := self.config.Clients[dockerID.Type]

    client := &Client{
        Config:     clientConfig,
        Instance:   dockerID.Instance,

        dockerID:   dockerID,
    }

    return client, nil
}

func (self *Manager) clientUp(config *ClientConfig, instance string) (*Client, error) {
    client := &Client{
        Config:     config,
        Instance:   instance,
        up:         true,

        dockerID:   docker.ID{Class:"client", Type: config.name, Instance: instance},
    }

    // docker
    dockerConfig := docker.Config{
        Image:      config.Image,
        Privileged: config.Privileged,
    }
    dockerConfig.Env.AddEnv("CLOSE_INSTANCE", client.String())

    if config.Volume != "" {
        bind := config.VolumePath
        if config.VolumeFmtID != "" {
            bind += fmt.Sprintf(config.VolumeFmtID, instance)
        }

        dockerConfig.AddMount(config.Volume, bind, config.VolumeReadonly)
    }

    if dockerContainer, err := self.docker.Up(client.dockerID, dockerConfig); err != nil {
        return nil, fmt.Errorf("docker.Up %v: %v", client, err)
    } else {
        self.log.Printf("docker.Up %v: %v", client, dockerContainer)
    }

    return client, nil
}

// Mark all clients as down
func (self *Manager) markClients() {
    for _, client := range self.clients {
        client.up = false
    }
}

// Start up all configured clients
func (self *Manager) ClientUp(config *ClientConfig) (errs []error) {
    self.log.Printf("ClientUp %v: Start %d clients...\n", config, config.Count)

    for index := uint(1); index <= config.Count; index++ {
        instance := fmt.Sprintf("%d", index)

        if client, err := self.clientUp(config, instance); err != nil {
            errs = append(errs, fmt.Errorf("ClientUp %v: %v", config, err))
        } else {
            self.clients[client.String()] = client
        }
    }

    return errs
}

// Stop any clients that are not configured to be up
func (self *Manager) sweepClients() (errs []error) {
    for _, client := range self.clients {
        if client.up {
            continue
        }
        if err := self.docker.Down(client.dockerID); err != nil {
            errs = append(errs, fmt.Errorf("sweepClients %v: docker.Down %v: %v", client, client.dockerID, err))
        }
    }

    return errs
}

// Stop running clients clients for given config
//
// Call with config=nil to stop all clients
func (self *Manager) ClientDown(config *ClientConfig) (errs []error) {
    for _, client := range self.clients {
        if config != nil && client.Config != config {
            continue
        }

        client.up = false

        if err := self.docker.Down(client.dockerID); err != nil {
            errs = append(errs, fmt.Errorf("ClientDown %v: docker.Down %v: %v", config, client.dockerID, err))
        }
    }

    return errs
}

// Cleanup down'd clients
func (self *Manager) ClientClean() (errs []error) {
    for key, client := range self.clients {
        if client.up {
            continue
        }

        if err := self.docker.Clean(client.dockerID); err != nil {
            errs = append(errs, fmt.Errorf("ClientClean %v: docker.Clean %v: %v", client, client.dockerID, err))
        }

        delete(self.clients, key)
    }

    return errs
}

func (self *Manager) GetClient(config string, instance string) (*Client, error) {
    clientName := fmt.Sprintf("%s:%v", config, instance)

    if client, exists := self.clients[clientName]; !exists {
        return nil, fmt.Errorf("Client not found: %v", clientName)
    } else {
        return client, nil
    }
}

type ClientState string

var ClientDown  ClientState = "down"
var ClientUp    ClientState = "up"
var ClientError ClientState = "error"

type ClientStatus struct {
    Config          string      `json:"config"`
    Instance        string      `json:"instance"`

    Docker          string      `json:"docker"`
    DockerStatus    string      `json:"docker_status"`
    DockerNode      string      `json:"docker_node"`

    Up              bool        `json:"up"`
    State           ClientState `json:"state"`
}

func (client *Client) GetStatus(dockerCache *docker.Cache) (ClientStatus, error) {
    clientStatus := ClientStatus{
        Config:         client.Config.name,
        Instance:       client.Instance,

        Up:             client.up,
    }

    if dockerStatus, err := dockerCache.GetStatus(client.dockerID); err != nil {
        clientStatus.Docker = client.dockerID.String()
        clientStatus.DockerStatus = fmt.Sprintf("%v", err)
        clientStatus.State = ClientError
    } else if dockerStatus == nil {
        clientStatus.Docker = ""
        clientStatus.DockerStatus = ""
        clientStatus.State = ClientDown
    } else {
        clientStatus.Docker = dockerStatus.ID.String()
        clientStatus.DockerStatus = dockerStatus.Status
        clientStatus.DockerNode = dockerStatus.Node

        if dockerStatus.IsUp() {
            clientStatus.State = ClientUp
        } else if dockerStatus.IsError() {
            clientStatus.State = ClientError
        } else {
            clientStatus.State = ClientDown
        }
    }

    return clientStatus, nil
}

func (self *Manager) ListClients() (clients []ClientStatus, err error) {
    dockerCache := self.docker.NewCache(true)

    for _, client := range self.clients {
        if clientStatus, err := client.GetStatus(dockerCache); err != nil {
            return nil, err
        } else {
            clients = append(clients, clientStatus)
        }
    }

    return clients, nil
}

func (self *Manager) ClientDelete(configName string, instance string) error {
    for key, client := range self.clients {
        if configName != "" && (client.Config == nil || client.Config.name != configName) {
            continue
        }
        if instance != "" && client.Instance != instance {
            continue
        }

        if err := self.docker.Clean(client.dockerID); err != nil {
            return err
        }

        delete(self.clients, key)
    }

    return nil
}
