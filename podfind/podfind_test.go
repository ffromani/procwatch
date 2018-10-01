package podfind

import "testing"

func TestNoCgroupData(t *testing.T) {
	containerID, cgroupStyle := parseProcCGroupEntry("/dev/null")
	if containerID != "" {
		t.Errorf("unexpected containerID: %v", containerID)
	}
	if cgroupStyle != MissingCGroup {
		t.Errorf("unexpected cgroupStyle: %v", cgroupStyle)
	}
}

func TestEmptyCgroupData(t *testing.T) {
	containerID, cgroupStyle := parseProcCGroupEntry("cgroup-empty.data")
	if containerID != "/" {
		t.Errorf("unexpected containerID: %v", containerID)
	}
	if cgroupStyle != UnknownCGroup {
		t.Errorf("unexpected cgroupStyle: %v", cgroupStyle)
	}
}

func TestEmptyDockerData(t *testing.T) {
	containerID, cgroupStyle := parseProcCGroupEntry("cgroup-docker.data")
	if containerID != "0fca315d4f86002639693ca3dbf16e4376a1951ea18ab537a38bb12478de161c" {
		t.Errorf("unexpected containerID: %v", containerID)
	}
	if cgroupStyle != DockerCGroup {
		t.Errorf("unexpected cgroupStyle: %v", cgroupStyle)
	}
}
