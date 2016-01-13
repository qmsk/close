package control

// A client is a docker container providing a networking environment for running workers in
// it does not provide any worker config/stats

import (
    "fmt"
)

type ClientConfig struct {
    Name            string
    Count           uint

    Image           string
    Privileged      bool

    Volume          string
    VolumePath      string
    VolumeFmtID     string
    VolumeReadonly  bool
}

type Client struct {
    Config      *ClientConfig

    Type        string
    ID          uint

    dockerContainer *DockerContainer
}

func (self Client) String() string {
    return fmt.Sprintf("%s:%d", self.Type, self.ID)
}

func (self *Manager) discoverClient(dockerContainer *DockerContainer) error {
    client := &Client{
        Config: nil, // TODO

        Type:   dockerContainer.Type,
        ID:     dockerContainer.Index,

        dockerContainer:    dockerContainer,
    }

    self.clients[client.String()] = client

    return nil
}

func (self *Manager) clientUp(config *ClientConfig, id uint) (*Client, error) {
    client := &Client{
        Config: config,
        Type:   config.Name,
        ID:     id,
    }

    // docker
    dockerID := DockerID{Class:"client", Type: config.Name, Index: id}

    dockerConfig := DockerConfig{
        Image:      config.Image,
        Env:        []string{
            fmt.Sprintf("CLOSE_ID=%d", id),
        },
        Privileged: config.Privileged,
    }

    if config.Volume != "" {
        bind := config.VolumePath
        if config.VolumeFmtID != "" {
            bind += fmt.Sprintf(config.VolumeFmtID, id)
        }

        dockerConfig.AddMount(config.Volume, bind, config.VolumeReadonly)
    }

    if container, err := self.DockerUp(dockerID, dockerConfig); err != nil {
        return nil, fmt.Errorf("DockerUp %v: %v", client, err)
    } else {
        client.dockerContainer = container
    }

    return client, nil
}

// Start up all configured clients
func (self *Manager) StartClients(config ClientConfig) error {
    // mark
    for _, client := range self.clients {
        client.Config = nil
    }

    self.log.Printf("Start %d %s clients...\n", config.Count, config.Name);

    for id := uint(1); id <= config.Count; id++ {
        if client, err := self.clientUp(&config, id); err != nil {
            return err
        } else {
            self.clients[client.String()] = client
        }
    }

    // sweep
    for key, client := range self.clients {
        if client.Config != nil {
            continue
        }

        if err := self.DockerDown(client.dockerContainer); err != nil {
            self.log.Printf("DockerDown %v: %v", client.dockerContainer, err)
        }

        delete(self.clients, key)
    }

    return nil
}

func (self *Manager) GetClient(name string, index uint) (*Client, error) {
    clientName := fmt.Sprintf("%s:%d", name, index)

    if client, exists := self.clients[clientName]; !exists {
        return nil, fmt.Errorf("Client not found: %v", clientName)
    } else {
        return client, nil
    }
}

// Stop all running clients
func (self *Manager) StopClients() (retErr error) {
    // sweep
    for key, client := range self.clients {
        if err := self.DockerDown(client.dockerContainer); err != nil {
            self.log.Printf("DockerDown %v: %v", client.dockerContainer, err)
            retErr = err
        }

        delete(self.clients, key)
    }

    return retErr
}

type ClientStatus struct {
    Config          string  `json:"config"`
    Type            string  `json:"type"`
    ID              uint    `json:"id"`

    Docker          string  `json:"docker"`
    DockerStatus    string  `json:"docker_status"`
}

func (self *Manager) ListClients() (clients []ClientStatus, err error) {
    for _, client := range self.clients {
        clientStatus := ClientStatus{
            Type:           client.Type,
            ID:             client.ID,
        }

        if client.Config != nil {
            clientStatus.Config = client.Config.Name
        }

        if dockerContainer, err := self.DockerGet(client.dockerContainer.String()); err != nil {
            return nil, err
        } else {
            clientStatus.Docker = dockerContainer.String()
            clientStatus.DockerStatus = dockerContainer.Status
        }

        clients = append(clients, clientStatus)
    }

    return clients, nil
}
