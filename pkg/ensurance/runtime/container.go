package runtime

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/net/context"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"

	"github.com/gocrane-io/crane/pkg/utils/clogs"
)

type UpdateOptions struct {
	// (Windows only) Number of CPUs available to the container.
	CPUCount int64
	// (Windows only) Portion of CPU cycles specified as a percentage * 100.
	CPUMaximum int64
	// CPU CFS (Completely Fair Scheduler) period. Default: 0 (not specified).
	CPUPeriod int64
	// CPU CFS (Completely Fair Scheduler) quota. Default: 0 (not specified).
	CPUQuota int64
	// CPU shares (relative weight vs. other containers). Default: 0 (not specified).
	CPUShares int64
	// Memory limit in bytes. Default: 0 (not specified).
	MemoryLimitInBytes int64
	// OOMScoreAdj adjusts the oom-killer score. Default: 0 (not specified).
	OomScoreAdj int64
	// CpusetCpus constrains the allowed set of logical CPUs. Default: "" (not specified).
	CpusetCpus string
	// CpusetMems constrains the allowed set of memory nodes. Default: "" (not specified).
	CpusetMems string
}

type ListOptions struct {
	// id of container or sandbox
	id string
	// podID of container
	podID string
	// Regular expression pattern to match pod or container
	nameRegexp string
	// Regular expression pattern to match the pod namespace
	podNamespaceRegexp string
	// state of the sandbox
	state string
	// show verbose info for the sandbox
	verbose bool
	// labels are selectors for the sandbox
	labels map[string]string
	// quiet is for listing just container/sandbox/image IDs
	quiet bool
	// output format
	output string
	// all containers
	all bool
	// latest container
	latest bool
	// last n containers
	last int
	// out with truncating the id
	noTrunc bool
	// image used by the container
	image string
}

type containerByCreated []*pb.Container

func (a containerByCreated) Len() int      { return len(a) }
func (a containerByCreated) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a containerByCreated) Less(i, j int) bool {
	return a[i].CreatedAt > a[j].CreatedAt
}

// UpdateContainerResources sends an UpdateContainerResourcesRequest to the server, and parses the returned UpdateContainerResourcesResponse.
func UpdateContainerResources(client pb.RuntimeServiceClient, containerId string, opts UpdateOptions) error {
	if containerId == "" {
		return fmt.Errorf("containerId cannot be empty")
	}
	request := &pb.UpdateContainerResourcesRequest{
		ContainerId: containerId,
		Linux: &pb.LinuxContainerResources{
			CpuPeriod:          opts.CPUPeriod,
			CpuQuota:           opts.CPUQuota,
			CpuShares:          opts.CPUShares,
			CpusetCpus:         opts.CpusetCpus,
			CpusetMems:         opts.CpusetMems,
			MemoryLimitInBytes: opts.MemoryLimitInBytes,
			OomScoreAdj:        opts.OomScoreAdj,
		},
	}

	clogs.Log().V(5).Info("UpdateContainerResourcesRequest: %v", request)
	r, err := client.UpdateContainerResources(context.Background(), request)
	if err != nil {
		return err
	}

	clogs.Log().V(5).Info("UpdateContainerResourcesResponse: %v", r)

	return nil
}

// RemoveContainer sends a RemoveContainerRequest to the server, and parses
// the returned RemoveContainerResponse.
func RemoveContainer(client pb.RuntimeServiceClient, ContainerId string) error {
	if ContainerId == "" {
		return fmt.Errorf("ID cannot be empty")
	}

	request := &pb.RemoveContainerRequest{
		ContainerId: ContainerId,
	}

	clogs.Log().V(5).Info("RemoveContainerRequest: %v", request)

	r, err := client.RemoveContainer(context.Background(), request)
	if err != nil {
		return err
	}

	clogs.Log().V(5).Info("RemoveContainerResponse: %v", r)
	return nil
}

// ListContainers sends a ListContainerRequest to the server, and parses the returned ListContainerResponse.
func ListContainers(runtimeClient pb.RuntimeServiceClient, opts ListOptions) ([]*pb.Container, error) {
	filter := &pb.ContainerFilter{}
	if opts.id != "" {
		filter.Id = opts.id
	}

	if opts.podID != "" {
		filter.PodSandboxId = opts.podID
	}

	st := &pb.ContainerStateValue{}
	if !opts.all && opts.state == "" {
		st.State = pb.ContainerState_CONTAINER_RUNNING
		filter.State = st
	}

	if opts.state != "" {
		st.State = pb.ContainerState_CONTAINER_UNKNOWN
		switch strings.ToLower(opts.state) {
		case "created":
			st.State = pb.ContainerState_CONTAINER_CREATED
			filter.State = st
		case "running":
			st.State = pb.ContainerState_CONTAINER_RUNNING
			filter.State = st
		case "exited":
			st.State = pb.ContainerState_CONTAINER_EXITED
			filter.State = st
		case "unknown":
			st.State = pb.ContainerState_CONTAINER_UNKNOWN
			filter.State = st
		default:
			clogs.Log().Error(fmt.Errorf("state should be one of created, running, exited or unknown"), "")
			return []*pb.Container{}, fmt.Errorf("state should be one of created, running, exited or unknown")
		}
	}

	if opts.latest || opts.last > 0 {
		// Do not filter by state if latest/last is specified.
		filter.State = nil
	}

	if opts.labels != nil {
		filter.LabelSelector = opts.labels
	}

	request := &pb.ListContainersRequest{
		Filter: filter,
	}

	clogs.Log().V(5).Info("ListContainerRequest: %v", request)

	r, err := runtimeClient.ListContainers(context.Background(), request)
	if err != nil {
		return []*pb.Container{}, err
	}

	clogs.Log().V(5).Info("ListContainerResponse: %v", r)

	r.Containers = filterContainersList(r.GetContainers(), opts)

	return r.Containers, nil
}

func filterContainersList(containersList []*pb.Container, opts ListOptions) []*pb.Container {
	var filtered = []*pb.Container{}

	for _, c := range containersList {
		if matched, err := regexp.MatchString(opts.nameRegexp, c.Metadata.Name); err == nil {
			if matched {
				filtered = append(filtered, c)
			}
		}
	}

	sort.Sort(containerByCreated(filtered))
	n := len(filtered)
	if opts.latest {
		n = 1
	}

	if opts.last > 0 {
		n = opts.last
	}

	n = func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}(n, len(filtered))

	return filtered[:n]
}
