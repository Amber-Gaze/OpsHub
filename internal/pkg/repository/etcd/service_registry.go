package etcd

type ServiceRegistry struct{}

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{}
}

func (r *ServiceRegistry) Register(name, addr string) error {
	return nil
}

func (r *ServiceRegistry) Discover(name string) ([]string, error) {
	return []string{}, nil
}
