package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"ldddns.arnested.dk/internal/container"
	"ldddns.arnested.dk/internal/hostname"
	"ldddns.arnested.dk/internal/log"
)

func handleContainer(
	ctx context.Context,
	docker *client.Client,
	containerID string,
	egs *EntryGroups,
	status string,
	config Config,
) error {
	eg, commit, err := egs.Get(containerID)
	defer commit()

	if err != nil {
		return fmt.Errorf("cannot get entry group for container: %w", err)
	}

	empty, err := eg.IsEmpty()
	if err != nil {
		return fmt.Errorf("checking whether Avahi entry group is empty: %w", err)
	}

	if !empty {
		err := eg.Reset()
		if err != nil {
			return fmt.Errorf("resetting Avahi entry group is empty: %w", err)
		}
	}

	if status == "die" || status == "kill" || status == "pause" {
		return nil
	}

	containerJSON, err := docker.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("inspecting container: %w", err)
	}

	c := container.Container{ContainerJSON: containerJSON}

	ips := c.IPAddresses()
	if len(ips) == 0 {
		return nil
	}

	hostnames, err := hostname.Hostnames(c, config.HostnameLookup)
	if err != nil {
		return fmt.Errorf("getting hostnames: %w", err)
	}

	for _, hostname := range hostnames {
		addAddress(eg, hostname, ips)
	}

	if services := c.Services(); len(hostnames) > 0 {
		addServices(eg, hostnames[0], ips, services, c.Name())
	}

	return nil
}

func handleExistingContainers(ctx context.Context, config Config, docker *client.Client, egs *EntryGroups) {
	containers, err := docker.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		log.Logf(log.PriErr, "getting container list: %v", err)
	}

	for _, container := range containers {
		err = handleContainer(ctx, docker, container.ID, egs, "start", config)
		if err != nil {
			log.Logf(log.PriErr, "handling container: %v", err)

			continue
		}
	}
}

func listen(ctx context.Context, config Config, docker *client.Client, egs *EntryGroups, started time.Time) {
	filter := filters.NewArgs()
	filter.Add("type", "container")
	filter.Add("event", "die")
	filter.Add("event", "kill")
	filter.Add("event", "pause")
	filter.Add("event", "start")
	filter.Add("event", "unpause")

	msgs, errs := docker.Events(ctx, types.EventsOptions{
		Filters: filter,
		Since:   strconv.FormatInt(started.Unix(), 10),
		Until:   "",
	})

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM)

	for {
		select {
		case err := <-errs:
			panic(fmt.Errorf("go error reading docker events: %w", err))
		case msg := <-msgs:
			err := handleContainer(ctx, docker, msg.ID, egs, msg.Status, config)
			if err != nil {
				log.Logf(log.PriErr, "handling container: %v", err)
			}
		case <-sig:
			log.Logf(log.PriNotice, "Shutting down")
			os.Exit(int(syscall.SIGTERM))
		}
	}
}
