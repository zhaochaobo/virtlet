/*
Copyright 2017 Mirantis

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"flag"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/gomega"

	"k8s.io/api/core/v1"

	"github.com/Mirantis/virtlet/tests/e2e/framework"
	. "github.com/Mirantis/virtlet/tests/e2e/ginkgo-ext"
)

var (
	vmImageLocation       = flag.String("image", defaultVMImageLocation, "VM image URL (*without http(s)://*")
	sshUser               = flag.String("sshuser", defaultSSHUser, "default SSH user for VMs")
	includeCloudInitTests = flag.Bool("include-cloud-init-tests", false, "include Cloud-Init tests")
	includeUnsafeTests    = flag.Bool("include-unsafe-tests", false, "include tests that can be unsafe if they're run outside the build container")
	memoryLimit           = flag.Int("memoryLimit", 160, "default VM memory limit (in MiB)")
	junitOutput           = flag.String("junitOutput", "", "JUnit XML output file")
)

// scheduleWaitSSH schedules SSH interface initialization before the test context starts
func scheduleWaitSSH(vm **framework.VMInterface, ssh *framework.Executor) {
	BeforeAll(func() {
		*ssh = waitSSH(*vm)
	})

	AfterAll(func() {
		(*ssh).Close()
	})
}

func waitSSH(vm *framework.VMInterface) framework.Executor {
	var ssh framework.Executor
	Eventually(
		func() error {
			var err error
			ssh, err = vm.SSH(*sshUser, sshPrivateKey)
			if err != nil {
				return err
			}
			_, err = framework.RunSimple(ssh)
			return err
		}, 60*5, 3).Should(Succeed())
	return ssh
}

func waitVirtletPod(vm *framework.VMInterface) *framework.PodInterface {
	var virtletPod *framework.PodInterface
	Eventually(
		func() error {
			var err error
			virtletPod, err = vm.VirtletPod()
			if err != nil {
				return err
			}
			for _, c := range virtletPod.Pod.Status.Conditions {
				if c.Type == v1.PodReady && c.Status == v1.ConditionTrue {
					return nil
				}
			}
			return fmt.Errorf("Pod not ready yet: %+v", virtletPod.Pod.Status)
		}, 60*5, 3).Should(Succeed())
	return virtletPod
}

func checkCPUCount(vm *framework.VMInterface, ssh framework.Executor, cpus int) {
	proc := do(framework.RunSimple(ssh, "cat", "/proc/cpuinfo")).(string)
	Expect(regexp.MustCompile(`(?m)^processor`).FindAllString(proc, -1)).To(HaveLen(cpus))
	cpuStats := do(vm.VirshCommand("domstats", "<domain>", "--vcpu")).(string)
	match := regexp.MustCompile(`vcpu\.maximum=(\d+)`).FindStringSubmatch(cpuStats)
	Expect(match).To(HaveLen(2))
	Expect(strconv.Atoi(match[1])).To(Equal(cpus))
}

func deleteVM(vm *framework.VMInterface) {
	virtletPod, err := vm.VirtletPod()
	Expect(err).NotTo(HaveOccurred())

	domainName, err := vm.DomainName()
	Expect(err).NotTo(HaveOccurred())
	domainName = domainName[8:21] // extract 5d3f8619-fda4 from virtlet-5d3f8619-fda4-cirros-vm

	Expect(vm.Delete(time.Minute * 2)).To(Succeed())

	commands := map[string][]string{
		"domain": {"list", "--name"},
		"volume": {"vol-list", "--pool", "volumes"},
		"secret": {"secret-list"},
	}

	for key, cmd := range commands {
		Eventually(func() error {
			out, err := framework.RunVirsh(virtletPod, cmd...)
			if err != nil {
				return err
			}
			if strings.Contains(out, domainName) {
				return fmt.Errorf("%s ~%s~ was not deleted", key, domainName)
			}
			return nil
		}, "3m").Should(Succeed())
	}
}

// do asserts that function with multiple return values doesn't fail
// considering we have func `foo(something) (something, error)`
//
// `x := do(foo(something))` is equivalent to
// val, err := fn(something)
// Expect(err).To(Succeed())
// x = val
func do(value interface{}, extra ...interface{}) interface{} {
	ExpectWithOffset(1, value, extra...).To(BeAnything())
	return value
}

type VMOptions framework.VMOptions

func (o VMOptions) applyDefaults() framework.VMOptions {
	res := framework.VMOptions(o)
	if res.Image == "" {
		res.Image = *vmImageLocation
	}
	if res.SSHKey == "" && res.SSHKeySource == "" {
		res.SSHKey = sshPublicKey
	}
	if res.VCPUCount == 0 {
		res.VCPUCount = 1
	}
	if res.DiskDriver == "" {
		res.DiskDriver = "virtio"
	}
	if res.Limits == nil {
		res.Limits = map[string]string{}
	}
	if res.Limits["memory"] == "" {
		res.Limits["memory"] = fmt.Sprintf("%dMi", *memoryLimit)
	}

	return res
}

func requireCloudInit() {
	if !*includeCloudInitTests {
		Skip("Cloud-Init tests are not enabled")
	}
}

func includeUnsafe() {
	if !*includeUnsafeTests {
		Skip("Tests that are unsafe outside the build container are disabled")
	}
}
