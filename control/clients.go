package control

// A client is a docker container providing a networking environment for running workers in
// it does not provide any worker config/stats

import (
    "fmt"
    "log"
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
    ID          uint

    dockerContainer *DockerContainer
}

func (self Client) String() string {
    return fmt.Sprintf("%s:%d", self.Config.Name, self.ID)
}

func (self *Manager) clientUp(config *ClientConfig, id uint) (*Client, error) {
    client := &Client{
        Config: config,
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
        log.Printf("DockerUP client %v: %v\n", client, container)

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
            log.Printf("DockerDown %v: %v", client.dockerContainer, err)
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
