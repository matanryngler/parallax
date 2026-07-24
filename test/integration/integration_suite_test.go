/*
Copyright 2025.

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

package integration

import (
	"fmt"
	"os/exec"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/matanryngler/parallax/test/utils"
)

var (
	postgresCmd *exec.Cmd
	mockApiCmd  *exec.Cmd
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting parallax integration test suite\n")
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func() {
	By("waiting for parallax deployment to be ready")
	err := utils.WaitForDeployment("parallax", "parallax-system", 300)
	Expect(err).NotTo(HaveOccurred())

	By("waiting for postgres to be ready")
	err = utils.WaitForDeployment("postgres", "default", 300)
	Expect(err).NotTo(HaveOccurred())

	By("waiting for mock-api to be ready")
	err = utils.WaitForDeployment("mock-api", "default", 300)
	Expect(err).NotTo(HaveOccurred())

	By("starting port-forward for postgres")
	postgresCmd, err = utils.PortForward("default", "postgres", 5432, 5432)
	Expect(err).NotTo(HaveOccurred())

	By("starting port-forward for mock-api")
	// The service port is 80 (mapping to 8080), so we forward local 8080 to remote 80
	mockApiCmd, err = utils.PortForward("default", "mock-api", 8080, 80)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("terminating port-forward processes")
	if postgresCmd != nil && postgresCmd.Process != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Terminating postgres port-forward (PID: %d)\n", postgresCmd.Process.Pid)
		_ = postgresCmd.Process.Kill()
	}
	if mockApiCmd != nil && mockApiCmd.Process != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Terminating mock-api port-forward (PID: %d)\n", mockApiCmd.Process.Pid)
		_ = mockApiCmd.Process.Kill()
	}

	By("cleaning up test namespaces")
	// Generic cleanup for any namespaces created during tests
	// This is a placeholder as specific test namespaces are not yet defined.
	// We can add logic here to delete namespaces with a specific label if needed.
	cmd := exec.Command("kubectl", "delete", "ns", "-l", "test=integration", "--ignore-not-found")
	_, _ = utils.Run(cmd)
})
