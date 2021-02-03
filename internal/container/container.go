package container

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"honnef.co/go/netdb"
)

type Container struct {
	types.ContainerJSON
}

// Names is the containers name without the leading '/'.
func (c Container) Name() string {
	return c.ContainerJSON.Name[1:]
}

func (c Container) IPAddresses() []string {
	ips := []string{}

	if c.NetworkSettings.IPAddress != "" {
		ips = append(ips, c.NetworkSettings.IPAddress)
	}

	for _, v := range c.NetworkSettings.Networks {
		ips = append(ips, v.IPAddress)
	}

	return ips
}

// Services from a container.
func (c Container) Services() map[string]uint16 {
	services := map[string]uint16{}

	for k := range c.NetworkSettings.Ports {
		port := strings.SplitN(string(k), "/", 2)

		proto := netdb.GetProtoByName(port[1])

		portNumber, err := strconv.ParseUint(port[0], 10, 16)
		if err != nil {
			log.Printf("Could not get port number from %q", k)

			continue
		}

		service := netdb.GetServByPort(int(portNumber), proto)

		if service == nil || proto == nil {
			continue
		}

		services[fmt.Sprintf("_%s._%s", service.Name, proto.Name)] = uint16(portNumber)
	}

	return services
}

// HostnamesFromEnv a container, return them as string slices.
func (c Container) HostnamesFromEnv(envName string) []string {
	prefix := envName + "="

	for _, s := range c.Config.Env {
		if strings.HasPrefix(s, prefix) {
			// Support multiple hostnames separated with comma and/or space.
			return strings.FieldsFunc(s[len(prefix):], func(r rune) bool { return r == ' ' || r == ',' })
		}
	}

	return []string{}
}
