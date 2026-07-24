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

var _ = Describe("Scenario 01: Static List", func() {
	const (
		namespace = "test-static-list"
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

	Context("with a basic static list", func() {
		It("should create Jobs for each item in the list", func() {
			projectDir, _ := utils.GetProjectDir()
			exampleDir := projectDir + "/examples/01-basic-static-list"

			By("applying the ListSource")
			cmd := exec.Command("kubectl", "apply", "-f", exampleDir+"/listsource.yaml", "-n", namespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("applying the ListJob")
			cmd = exec.Command("kubectl", "apply", "-f", exampleDir+"/listjob.yaml", "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the ListSource to be ready")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "listsource", "basic-list", "-n", namespace, "-o", "jsonpath={.status.state}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(Equal("Ready"))

			By("waiting for the Job to complete")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "job", "basic-processor", "-n", namespace, "-o", "jsonpath={.status.conditions[?(@.type==\"Complete\")].status}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(Equal("True"))

			By("verifying that the Job succeeded with 3 completions")
			cmd = exec.Command("kubectl", "get", "job", "basic-processor", "-n", namespace, "-o", "jsonpath={.status.succeeded}")
			output, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(Equal("3"))

			By("verifying that each Pod succeeded")
			for i := 0; i < 3; i++ {
				Eventually(func() string {
					cmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "-l", "listjob.batchops.io/name=basic-processor", fmt.Sprintf("-o=jsonpath={.items[%d].status.phase}", i))
					output, _ := utils.Run(cmd)
					return output
				}, timeout, interval).Should(Equal("Succeeded"))
			}
		})
	})
})
