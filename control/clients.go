package control

// A client is a docker container providing a networking environment for running workers in
// it does not provide any worker config/stats

import (
    "close/docker"
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

    dockerID        docker.ID
}

func (self Client) String() string {
    return fmt.Sprintf("%v:%s", self.Config, self.Instance)
}

func (self *Manager) discoverClient(dockerID docker.ID) (*Client, error) {
    clientConfig := self.config.Clients[dockerID.Type]

    if clientConfig == nil {
        return nil, fmt.Errorf("Unknown client config type: %v", dockerID.Type)
    }

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

        dockerID:   docker.ID{Class:"client", Type: config.name, Instance: instance},
    }

    // docker
    dockerConfig := docker.Config{
        Image:      config.Image,
        Privileged: config.Privileged,
    }
    dockerConfig.Env.Add("CLOSE_INSTANCE", client.String())

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

// Mark all clients as unconfigured
func (self *Manager) markClients() {
    for _, client := range self.clients {
        client.Config = nil
    }
}

// Start up all configured clients
func (self *Manager) ClientUp(config *ClientConfig) error {
    self.log.Printf("ClientUp %v: Start %d clients...\n", config, config.Count)

    for index := uint(1); index <= config.Count; index++ {
        instance := fmt.Sprintf("%d", index)

        if client, err := self.clientUp(config, instance); err != nil {
            return fmt.Errorf("ClientUp %v: %v", config, err)
        } else {
            self.clients[client.String()] = client
        }
    }

    return nil
}

// Stop running clients clients for given config
// Call with config=nil to cleanup all unconfigured clients
func (self *Manager) ClientDown(config *ClientConfig) error {
    for key, client := range self.clients {
        if client.Config == config {
            if err := self.docker.Down(client.dockerID); err != nil {
                return fmt.Errorf("ClientDown %v: docker.Down %v: %v", config, client.dockerID, err)
            }

            delete(self.clients, key)
        }
    }

    return nil
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

    State           ClientState `json:"state"`
}

func (self *Manager) ListClients() (clients []ClientStatus, err error) {
    for _, client := range self.clients {
        clientStatus := ClientStatus{
            Config:         client.Config.name,
            Instance:       client.Instance,
        }

        if dockerContainer, err := self.docker.Get(client.dockerID.String()); err != nil {
            return nil, err
        } else if dockerContainer == nil {
            clientStatus.Docker = ""
            clientStatus.DockerStatus = ""
            clientStatus.State = ClientDown

        } else {
            clientStatus.Docker = dockerContainer.String()
            clientStatus.DockerStatus = dockerContainer.Status

            if dockerContainer.State.Running {
                clientStatus.State = ClientUp
            } else if dockerContainer.State.ExitCode == 0 {
                clientStatus.State = ClientDown
            } else {
                clientStatus.State = ClientError
            }
        }

        clients = append(clients, clientStatus)
    }

    return clients, nil
}
