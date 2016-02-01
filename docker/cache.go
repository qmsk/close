package docker

type Cache struct {
    manager         *Manager
    eager           bool

    // lookups already performed, used for negative cache
    list            map[ID]bool

    // lookup result, used for positive cache
    containerStatus map[ID]ContainerStatus
}

func (manager *Manager) NewCache(eager bool) *Cache {
    return &Cache{
        manager:    manager,
        eager:      eager,

        list:               make(map[ID]bool),
        containerStatus:    make(map[ID]ContainerStatus),
    }
}

/*
 * Get full container state.
 */
func (cache *Cache) GetContainer(id ID) (*Container, error) {
    if container, err := cache.manager.Get(id.String()); err != nil {
        return nil, err
    } else if container == nil {
        return nil, nil
    } else {
        // populate status cache aswell
        cache.containerStatus[container.ID] = container.ContainerStatus

        return container, nil
    }
}

/*
 * Return docker container status, or nil if not exists.
 *
 * Cached for all containers using list if eager.
 */
func (cache *Cache) GetStatus(id ID) (*ContainerStatus, error) {
    if containerStatus, exists := cache.containerStatus[id]; exists {
        return &containerStatus, nil
    }

    // lookup?
    filter := id

    if cache.eager {
        // all containers
        filter = ID{Class: id.Class}
    }

    if cached := cache.list[filter]; cached {
        // already did that query, negative result
        return nil, nil
    }

    // warm up cache
    if dockerList, err := cache.manager.List(filter); err != nil {
        return nil, err
    } else {
        for _, containerStatus := range dockerList {
            cache.containerStatus[containerStatus.ID] = containerStatus
        }

        cache.list[filter] = true
    }

    // return from cache
    if containerStatus, exists := cache.containerStatus[id]; exists {
        return &containerStatus, nil
    } else {
        return nil, nil
    }
}

