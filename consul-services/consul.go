package consul

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/CiscoCloud/mesos-consul/registry"

	log "github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
)

type Consul struct {
	agents map[string]*consulapi.Client
	config consulConfig
}

//
func New() *Consul {
	return &Consul{
		agents: make(map[string]*consulapi.Client),
		config: config,
	}
}

// client()
//   Return a consul client at the specified address
func (c *Consul) client(address string) *consulapi.Client {
	if address == "" {
		log.Warn("No address to Consul.Agent")
		return nil
	}

	if _, ok := c.agents[address]; !ok {
		// Agent connection not saved. Connect.
		c.agents[address] = c.newAgent(address)
	}

	return c.agents[address]
}

// newAgent()
//   Connect to a new agent specified by address
//
func (c *Consul) newAgent(address string) *consulapi.Client {
	if address == "" {
		log.Warnf("No address to Consul.NewAgent")
		return nil
	}

	config := consulapi.DefaultConfig()

	config.Address = fmt.Sprintf("%s:%s", address, c.config.port)
	log.Debugf("consul address: %s", config.Address)

	if c.config.token != "" {
		log.Debugf("setting token to %s", c.config.token)
		config.Token = c.config.token
	}

	if c.config.sslEnabled {
		log.Debugf("enabling SSL")
		config.Scheme = "https"
	}

	if !c.config.sslVerify {
		log.Debugf("disabled SSL verification")
		config.HttpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	if c.config.auth.Enabled {
		log.Debugf("setting basic auth")
		config.HttpAuth = &consulapi.HttpBasicAuth{
			Username: c.config.auth.Username,
			Password: c.config.auth.Password,
		}
	}

	client, err := consulapi.NewClient(config)
	if err != nil {
		log.Fatal("consul: ", address)
	}
	return client
}

func (c *Consul) Register(service *registry.Service) {
	if _, ok := c.agents[service.Agent]; !ok {
		// Agent connection not saved. Connect.
		c.agents[service.Agent] = c.newAgent(service.Agent)
	}

	log.Info("Registering ", service.ID)

	s := &consulapi.AgentServiceRegistration{
		ID:      service.ID,
		Name:    service.Name,
		Port:    service.Port,
		Address: service.Address,
		Check: &consulapi.AgentServiceCheck{
			TTL:      service.Check.TTL,
			Script:   service.Check.Script,
			HTTP:     service.Check.HTTP,
			Interval: service.Check.Interval,
		},
	}

	if len(service.Tags) > 0 {
		s.Tags = service.Tags
	}

	err := c.agents[service.Agent].Agent().ServiceRegister(s)
	if err != nil {
		log.Warnf("Unable to register %s: %s", s.ID, err.Error())
		return
	}
}

func (c *Consul) Deregister(serviceId string, agent string) {
	if _, ok := c.agents[agent]; !ok {
		// Agent connection not saved. Connect.
		c.agents[agent] = c.newAgent(agent)
	}

	log.Info("Deregistering ", serviceId)

	err := c.agents[agent].Agent().ServiceDeregister(serviceId)
	if err != nil {
		log.Warnf("Unable to deregister %s: %s", serviceId, err.Error())
		return
	}
}

func (c *Consul) deregister(agent string, service *consulapi.AgentServiceRegistration) error {
	if _, ok := c.agents[agent]; !ok {
		// Agent connection not saved. Connect.
		c.agents[agent] = c.newAgent(agent)
	}

	return c.agents[agent].Agent().ServiceDeregister(service.ID)
}