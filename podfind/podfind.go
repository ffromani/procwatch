package podfind

import (
	"google.golang.org/grpc"
	pb "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	"k8s.io/kubernetes/pkg/kubelet/util"

	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func getAddressAndDialer(endpoint string) (string, func(addr string, timeout time.Duration) (net.Conn, error), error) {
	return util.GetAddressAndDialer(endpoint)
}

type PodResolver struct {
	conn           *grpc.ClientConn
	client         pb.RuntimeServiceClient
	containerToPod map[string]string
	podInfos       map[string]string
	Debug          bool
}

func NewPodResolver(runtimeEndPoint string, timeout time.Duration) (*PodResolver, error) {
	pr := &PodResolver{
		Debug: true,
	}

	addr, dialer, err := getAddressAndDialer(runtimeEndPoint)
	if err != nil {
		return nil, err
	}

	pr.conn, err = grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(timeout), grpc.WithDialer(dialer))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	pr.client = pb.NewRuntimeServiceClient(pr.conn)
	return pr, nil
}

func (pr *PodResolver) Update() error {
	var err error
	err = pr.updateInfoContainers()
	if err != nil {
		return err
	}
	return pr.updateInfoPods()
}

func (pr *PodResolver) updateInfoContainers() error {
	st := &pb.ContainerStateValue{}
	st.State = pb.ContainerState_CONTAINER_RUNNING
	filter := &pb.ContainerFilter{}
	filter.State = st

	request := &pb.ListContainersRequest{
		Filter: filter,
	}

	r, err := pr.client.ListContainers(context.Background(), request)
	if err != nil {
		return err
	}

	pr.containerToPod = make(map[string]string)
	for _, c := range r.GetContainers() {
		pr.containerToPod[c.Id] = c.PodSandboxId
		if pr.Debug {
			fmt.Fprintf(os.Stderr, "CNT: %v -> %v\n", c.Id, pr.containerToPod[c.Id])
		}
	}

	return nil
}

func (pr *PodResolver) updateInfoPods() error {
	st := &pb.PodSandboxStateValue{}
	st.State = pb.PodSandboxState_SANDBOX_READY
	filter := &pb.PodSandboxFilter{}
	filter.State = st

	request := &pb.ListPodSandboxRequest{
		Filter: filter,
	}

	r, err := pr.client.ListPodSandbox(context.Background(), request)

	if err != nil {
		return err
	}

	pr.podInfos = make(map[string]string)
	for _, p := range r.GetItems() {
		if domainName, ok := p.Annotations["kubevirt.io/domain"]; ok {
			pr.podInfos[p.Id] = domainName
		} else {
			pr.podInfos[p.Id] = p.Metadata.Name
		}
		if pr.Debug {
			fmt.Fprintf(os.Stderr, "POD: %v -> %v\n", p.Id, pr.podInfos[p.Id])
		}
	}

	return nil
}

func (pr *PodResolver) FindPodByPID(pid int32) (string, error) {
	containerId, cgroupStyle := FindContainerIDByCGroup(pid)
	if cgroupStyle != DockerCGroup {
		return "", errors.New(fmt.Sprintf("unsupported cgroup style: %v", cgroupStyle))
	}
	podId, ok := pr.containerToPod[containerId]
	if !ok {
		return "", errors.New(fmt.Sprintf("no POD found for pid %v on container %v", pid, containerId))
	}
	podName, ok := pr.podInfos[podId]
	if !ok {
		return "", errors.New(fmt.Sprintf("no info for pid %v on container %v on pod %v", pid, containerId, podId))
	}
	return podName, nil
}

const (
	MissingCGroup = iota
	MalformedCGroup
	UnknownCGroup
	DockerCGroup
)

func FindContainerIDByCGroup(pid int32) (string, int) {
	return parseProcCGroupEntry(fmt.Sprintf("/proc/%d/cgroup", pid))
}

func parseProcCGroupEntry(entry string) (string, int) {
	file, err := os.Open(entry)
	defer file.Close()

	if err != nil {
		return "", MissingCGroup
	}

	reader := bufio.NewReader(file)
	line, isPrefix, err := reader.ReadLine()
	if isPrefix || err != nil {
		return "", MissingCGroup
	}

	return parseProcCGroupLine(string(line))
}

func parseProcCGroupLine(line string) (string, int) {
	fields := strings.Split(line, ":")
	// per cgroups(7):
	// hierarchy-ID:controller-list:cgroup-path
	if len(fields) != 3 {
		return "", MalformedCGroup
	}
	name := filepath.Base(fields[2])
	if strings.HasPrefix(name, "docker-") {
		name = name[len("docker-"):]
		if strings.HasSuffix(name, ".scope") {
			name = name[:len(name)-len(".scope")]
		}
		return name, DockerCGroup
	}
	return name, UnknownCGroup
}
