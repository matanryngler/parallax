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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/matanryngler/parallax/test/utils"
)

var _ = Describe("Scenario 02: API Integration", func() {
	const (
		namespace = "test-api-integration"
		timeout   = time.Second * 120
		interval  = time.Second * 2
	)

	BeforeEach(func() {
		By("creating the test namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, _ = utils.Run(cmd)

		By("labeling the namespace")
		cmd = exec.Command("kubectl", "label", "ns", namespace, "test=integration")
		_, _ = utils.Run(cmd)
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			By("debugging the namespace on failure")
			utils.DebugNamespace(namespace)
		}
		By("deleting the test namespace")
		cmd := exec.Command("kubectl", "delete", "ns", namespace, "--ignore-not-found")
		_, _ = utils.Run(cmd)
	})

	Context("with an API-based list source", func() {
		It("should fetch items from the API and create Jobs", func() {
			projectDir, _ := utils.GetProjectDir()
			exampleDir := projectDir + "/examples/02-api-integration"

			By("applying the ListSource (with patched URL)")
			// Patch url: http://mock-api/ to url: http://mock-api.default.svc.cluster.local/
			cmd := exec.Command("sh", "-c", fmt.Sprintf("sed 's|url: http://mock-api/|url: http://mock-api.default.svc.cluster.local/|' %s/listsource-api.yaml | kubectl apply -n %s -f -", exampleDir, namespace))
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("applying the ListJob")
			cmd = exec.Command("kubectl", "apply", "-f", exampleDir+"/listjob-api.yaml", "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the ListSource to be ready")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "listsource", "api-items", "-n", namespace, "-o", "jsonpath={.status.phase}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(Equal("Ready"))

			By("verifying that items were fetched")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "listsource", "api-items", "-n", namespace, "-o", "jsonpath={.status.itemCount}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(Equal("5"))

			By("waiting for the ListJob to complete")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "listjob", "api-processor", "-n", namespace, "-o", "jsonpath={.status.phase}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(Equal("Completed"))

			By("verifying that 5 Jobs were created and succeeded")
			cmd = exec.Command("kubectl", "get", "jobs", "-n", namespace, "-l", "listjob.batchops.io/name=api-processor", "--no-headers")
			output, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			lines := utils.GetNonEmptyLines(output)
			Expect(len(lines)).To(Equal(5))

			By("verifying that each Job succeeded")
			for i := 0; i < 5; i++ {
				Eventually(func() string {
					cmd := exec.Command("kubectl", "get", "jobs", "-n", namespace, "-l", "listjob.batchops.io/name=api-processor", fmt.Sprintf("-o=jsonpath={.items[%d].status.succeeded}", i))
					output, _ := utils.Run(cmd)
					return output
				}, timeout, interval).Should(Equal("1"))
			}
		})
	})
})
