package nodelocal

import (
	"flag"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"

	info "github.com/google/cadvisor/info/v1"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
)

var (
	// Define global args flags.
	//runtimeEndpoint      = flag.String("runtimeEndpoint", "unix:///var/run/dockershim.sock", "The runtime endpoint, default: unix:///var/run/dockershim.sock.")
	//runtimeEndpointIsSet = flag.Bool("runtimeEndpointIsSet", true, "The runtime endpoint set, default: true")
	//imageEndpoint        = flag.String("imageEndpoint", "unix:///var/run/dockershim.sock", "The image endpoint, default: unix:///var/run/dockershim.sock.")
	//imageEndpointIsSet   = flag.Bool("imageEndpointIsSet", true, "The image endpoint set, default: true")
	cgroupPath  = flag.String("cgroupPath", "", "The cgroup path, default: \"\"")
	containerId = flag.String("containerId", "", "The container id, default: \"\"")
)

func TestCadvisor(t *testing.T) {
	flag.Parse()

	//t.Logf("TestInitRuntimeClient runtimeEndpoint %v, runtimeEndpointIsSet %v, containerId %v", runtimeEndpoint, runtimeEndpointIsSet, containerId)

	c, err := NewCadvisorManager(nil)
	if err != nil {
		t.Fatalf("NewCadvisor failed %s", err.Error())
	}

	//c.Start()
	defer c.Stop()

	t.Logf("cadvisor %v", c)

	/*containerInfos, err := c.Manager.AllDockerContainers(nil)
	if err != nil {
		t.Fatalf("AllDockerContainers failed %s", err.Error())
	}
	t.Logf("containerInfos %v", containerInfos)*/

	if *cgroupPath != "" {
		var start = time.Now()
		containerInfo, err := c.Manager.GetContainerInfoV2(*cgroupPath, cadvisorapiv2.RequestOptions{
			IdType:    cadvisorapiv2.TypeName,
			Count:     2,
			Recursive: true,
		})

		var end = time.Now()

		if err != nil {
			t.Fatalf("GetContainerInfoV2 failed %s", err.Error())
		}

		//t.Logf("containerInfo %#v", containerInfo)
		t.Logf("cost:%d", end.UnixNano()-start.UnixNano())
		t.Logf("containerInfo Schedstat %#v", spew.Sdump(containerInfo[*cgroupPath].Stats[0].Cpu.Schedstat))
		t.Logf("containerInfo cpu %#v", spew.Sdump(containerInfo[*cgroupPath].Stats[0].Cpu.Usage.Total))
		t.Logf("containerInfo CpuSpec %#v", containerInfo)

		time.Sleep(1 * time.Second)

		var start1 = time.Now()

		var query = info.ContainerInfoRequest{NumStats: 60}

		containerInfo11, err := c.Manager.GetContainerInfo(*cgroupPath, &query)

		var end1 = time.Now()
		t.Logf("cost1:%d", end1.UnixNano()-start1.UnixNano())

		t.Logf("containerInfo11 CpuSpec %+v", containerInfo11.Spec.Cpu)

		/*info, err := c.Manager.GetContainerInfo(*cgroupPath,nil)
		if err != nil {
			t.Fatalf("GetContainerInfo failed %s", err.Error())
		}

		t.Logf("containerInfo CpuSpec %#v", spew.Sdump(info.Spec.Cpu))*/

		return

		containerInfo2, err := c.Manager.GetContainerInfoV2(*cgroupPath, cadvisorapiv2.RequestOptions{
			IdType:    cadvisorapiv2.TypeName,
			Count:     1,
			Recursive: true,
		})

		if err != nil {
			t.Fatalf("GetContainerInfoV2 failed %s", err.Error())
		}

		for key, v := range containerInfo2 {
			t.Logf("key: %#v", key)
			t.Logf("v: %#v", v)
		}

		var timeCost = containerInfo2[*cgroupPath].Stats[0].Timestamp.UnixNano() - containerInfo[*cgroupPath].Stats[0].Timestamp.UnixNano()
		var valueIncrease = containerInfo2[*cgroupPath].Stats[0].Cpu.Schedstat.RunqueueTime - containerInfo[*cgroupPath].Stats[0].Cpu.Schedstat.RunqueueTime
		var valueCpu = containerInfo2[*cgroupPath].Stats[0].Cpu.Usage.Total - containerInfo[*cgroupPath].Stats[0].Cpu.Usage.Total

		var per = uint64(float64(valueIncrease) / (float64(timeCost) / 1000.0 / 1000.0 / 1000.0))
		var cpuper = uint64((float64(valueCpu) / (float64(timeCost))) * 1000)

		t.Logf("per %d", per)
		t.Logf("cpuper %d", cpuper)
	}

	if *containerId != "" {
		containerInfo2, err := c.Manager.GetContainerInfoV2(*containerId, cadvisorapiv2.RequestOptions{
			IdType:    cadvisorapiv2.TypeDocker,
			Count:     1,
			Recursive: false,
		})

		if err != nil {
			t.Fatalf("GetContainerInfoV2 failed %s", err.Error())
		}

		t.Logf("containerInfo2 %#v", containerInfo2)
	}

	t.Logf("TestCadvisor succeed")
}

type KeyTest struct {
	input  string
	output string
}

func TestGetContainerIdFromKey(t *testing.T) {
	var cases = []KeyTest{
		{
			input:  "/kubepods/besteffort/pod04e5e9e7-8d95-44dd-9af7-ab944405fff8/18b514fc91ecb19b7ee79ebeaa6f2df86c6c939e420520b97ad4f7532582d35a",
			output: "18b514fc91ecb19b7ee79ebeaa6f2df86c6c939e420520b97ad4f7532582d35a",
		},
		{
			input:  "/kubepods/besteffort/pod04e5e9e7-8d95-44dd-9af7-ab944405fff8",
			output: "",
		},
		{
			input:  "/kubepods/besteffort/pod04e5e9e7-8d95-44dd-9af7-ab944405fff8/2cc2c4badac0618edda11bdd06826e7385b885ca88323b6f5d90270395e039d9",
			output: "2cc2c4badac0618edda11bdd06826e7385b885ca88323b6f5d90270395e039d9",
		},
	}

	for _, c := range cases {
		if r := GetContainerIdFromKey(c.input); r != c.output {
			t.Fatalf("TestGetContainerIdFromKey failed {%s,%s}, r: %s", c.input, c.output, r)
		}
	}

	return
}
