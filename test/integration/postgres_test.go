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

var _ = Describe("Scenario 03: Postgres ETL", func() {
	const (
		namespace = "test-postgres-etl"
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

	Context("with a postgres-based list source", func() {
		It("should fetch items from postgres and create Jobs", func() {
			projectDir, _ := utils.GetProjectDir()
			exampleDir := projectDir + "/examples/03-postgres-etl"

			By("applying the ListSource (with patched connectionString)")
			// Patch connectionString: postgresql://postgres:postgres@postgres:5432/testdb?sslmode=disable
			// to connectionString: postgresql://postgres:postgres@postgres.default.svc.cluster.local:5432/testdb?sslmode=disable
			cmd := exec.Command("sh", "-c", fmt.Sprintf("sed 's|@postgres:5432|@postgres.default.svc.cluster.local:5432|' %s/listsource-postgres.yaml | kubectl apply -n %s -f -", exampleDir, namespace))
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("applying the ListJob with the test infrastructure database host")
			cmd = exec.Command("sh", "-c", fmt.Sprintf("sed '/name: DB_HOST/{n; s|value: \"postgres\"|value: \"postgres.default.svc.cluster.local\"|;}' %s/listjob-etl.yaml | kubectl apply -n %s -f -", exampleDir, namespace))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the ListSource to be ready")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "listsource", "postgres-orders", "-n", namespace, "-o", "jsonpath={.status.state}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(Equal("Ready"))

			By("verifying that items were fetched")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "listsource", "postgres-orders", "-n", namespace, "-o", "jsonpath={.status.itemCount}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(Equal("5"))

			By("waiting for the Job to complete")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "job", "order-processor", "-n", namespace, "-o", "jsonpath={.status.conditions[?(@.type==\"Complete\")].status}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(Equal("True"))

			By("verifying that the Job succeeded with 5 completions")
			cmd = exec.Command("kubectl", "get", "job", "order-processor", "-n", namespace, "-o", "jsonpath={.status.succeeded}")
			output, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(Equal("5"))

			By("verifying that each Pod succeeded")
			for i := 0; i < 5; i++ {
				Eventually(func() string {
					cmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "-l", "listjob.batchops.io/name=order-processor", fmt.Sprintf("-o=jsonpath={.items[%d].status.phase}", i))
					output, _ := utils.Run(cmd)
					return output
				}, timeout, interval).Should(Equal("Succeeded"))
			}
		})
	})
})
