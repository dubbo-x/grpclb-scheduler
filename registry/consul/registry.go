package consul

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	consul "github.com/hashicorp/consul/api"
	"google.golang.org/grpc/grpclog"
)

type ConsulRegistry struct {
	ctx     context.Context
	cancel  context.CancelFunc
	client  *consul.Client
	cfg     *Congfig
	checkId string
}

type Congfig struct {
	ConsulCfg   *consul.Config
	ServiceName string
	NData       NodeData
	TTL         int // ttl seconds
}

type NodeData struct {
	ID       string
	Address  string
	Port     int
	Metadata map[string]string
}

func NewRegistry(cfg *Congfig) (*ConsulRegistry, error) {
	c, err := consul.NewClient(cfg.ConsulCfg)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())

	return &ConsulRegistry{
		ctx:     ctx,
		cancel:  cancel,
		client:  c,
		cfg:     cfg,
		checkId: "service:" + cfg.NData.ID,
	}, nil
}

func (c *ConsulRegistry) Register() error {

	// register service
	metadata, err := json.Marshal(c.cfg.NData.Metadata)
	if err != nil {
		return err
	}
	tags := make([]string, 0)
	tags = append(tags, string(metadata))

	registerHandler := func() error {
		regisDto := &consul.AgentServiceRegistration{
			ID:      c.cfg.NData.ID,
			Name:    c.cfg.ServiceName,
			Address: c.cfg.NData.Address,
			Port:    c.cfg.NData.Port,
			Tags:    tags,
			Check: &consul.AgentServiceCheck{
				TTL:                            fmt.Sprintf("%ds", c.cfg.TTL),
				Status:                         consul.HealthPassing,
				DeregisterCriticalServiceAfter: "1m",
			},
		}
		err := c.client.Agent().ServiceRegister(regisDto)
		if err != nil {
			return fmt.Errorf("register service to consul error: %s\n", err.Error())
		}

		return nil
	}

	err = registerHandler()
	if err != nil {
		return err
	}

	keepAliveTicker := time.NewTicker(time.Duration(c.cfg.TTL) * time.Second / 3)
	registerTicker := time.NewTicker(time.Minute)

	for {
		select {
		case <-c.ctx.Done():
			keepAliveTicker.Stop()
			registerTicker.Stop()
			c.client.Agent().ServiceDeregister(c.cfg.NData.ID)
			return nil

		case <-keepAliveTicker.C:
			err := c.client.Agent().PassTTL(c.checkId, "")
			if err != nil {
				grpclog.Printf("consul registry check %v.\n", err)
			}

		case <-registerTicker.C:
			err = registerHandler()
			if err != nil {
				grpclog.Printf("consul register service error: %v.\n", err)
			}
		}
	}
}

func (c *ConsulRegistry) Deregister() error {
	c.cancel()
	return nil
}
