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

var _ = Describe("Scenario 04: Scheduled Batch", func() {
	const (
		namespace = "test-scheduled-batch"
		timeout   = time.Second * 180 // Increased timeout for cron
		interval  = time.Second * 5
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

	Context("with a scheduled batch (ListCronJob)", func() {
		It("should create ListJobs based on the schedule", func() {
			projectDir, _ := utils.GetProjectDir()
			exampleDir := projectDir + "/examples/04-scheduled-batch"

			By("applying the ListSource")
			cmd := exec.Command("kubectl", "apply", "-f", exampleDir+"/listsource.yaml", "-n", namespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("applying the ListCronJob (with faster schedule)")
			// Patch schedule: "*/5 * * * *" to "* * * * *"
			cmd = exec.Command("sh", "-c", fmt.Sprintf("sed 's|schedule: \"\\*/5 \\* \\* \\* \\*\"|schedule: \"* * * * *\"|' %s/listcronjob.yaml | kubectl apply -n %s -f -", exampleDir, namespace))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the ListCronJob to be active")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "listcronjob", "report-generator", "-n", namespace, "-o", "jsonpath={.status.lastScheduleTime}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).ShouldNot(BeEmpty())

			By("verifying that a ListJob was created")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "listjobs", "-n", namespace, "-l", "listcronjob.batchops.io/name=report-generator", "--no-headers")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).ShouldNot(BeEmpty())

			By("waiting for at least one ListJob to complete")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "listjobs", "-n", namespace, "-l", "listcronjob.batchops.io/name=report-generator", "-o", "jsonpath={.items[0].status.phase}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(Equal("Completed"))

			By("verifying that Jobs were created for the items")
			cmd = exec.Command("kubectl", "get", "listjobs", "-n", namespace, "-l", "listcronjob.batchops.io/name=report-generator", "-o", "jsonpath={.items[0].metadata.name}")
			listJobName, _ := utils.Run(cmd)
			
			cmd = exec.Command("kubectl", "get", "jobs", "-n", namespace, "-l", "listjob.batchops.io/name="+listJobName, "--no-headers")
			output, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			lines := utils.GetNonEmptyLines(output)
			Expect(len(lines)).To(Equal(5))
		})
	})
})
