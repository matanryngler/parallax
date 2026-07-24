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
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/matanryngler/parallax/test/utils"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Scenario 05: Production Patterns", func() {
	const (
		namespace = "test-production-patterns"
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

	Context("with production security and resource patterns", func() {
		It("should propagate security context and resources to Job pods", func() {
			projectDir, _ := utils.GetProjectDir()
			exampleDir := projectDir + "/examples/05-production-patterns"

			By("creating a dummy ListSource for the reference")
			// Scenario 05 refers to 'production-data'
			rawCM := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: production-data
data:
  items: "item-1\nitem-2"
`
			cmd := exec.Command("sh", "-c", fmt.Sprintf("echo '%s' | kubectl apply -f - -n %s", rawCM, namespace))
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("creating the required ServiceAccount")
			rawSA := `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: parallax-job-runner
`
			cmd = exec.Command("sh", "-c", fmt.Sprintf("echo '%s' | kubectl apply -f - -n %s", rawSA, namespace))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("creating the Secrets referenced by the production template")
			rawSecrets := `
apiVersion: v1
kind: Secret
metadata:
  name: api-credentials
stringData:
  token: test-token
---
apiVersion: v1
kind: Secret
metadata:
  name: db-credentials
stringData:
  password: test-password
`
			cmd = exec.Command("sh", "-c", fmt.Sprintf("echo '%s' | kubectl apply -f - -n %s", rawSecrets, namespace))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("applying the ListJob with security and resources")
			cmd = exec.Command("sh", "-c", fmt.Sprintf("sed -e '/namespace: production/d' -e 's|image: your-registry/processor:v1.2.3|image: busybox:1.36|' %s/security-context.yaml | kubectl apply -n %s -f -", exampleDir, namespace))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the Job to be created")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "jobs", "-n", namespace, "-l", "listjob.batchops.io/name=secure-processor", "-o", "jsonpath={.items[0].metadata.name}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).ShouldNot(BeEmpty())

			By("verifying the pod security context")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "-l", "listjob.batchops.io/name=secure-processor", "-o", "json")
				output, _ := utils.Run(cmd)

				var podList corev1.PodList
				if err := json.Unmarshal([]byte(output), &podList); err != nil || len(podList.Items) == 0 {
					return false
				}

				pod := podList.Items[0]
				sc := pod.Spec.SecurityContext
				if sc == nil {
					return false
				}

				return *sc.RunAsUser == 1000 && *sc.RunAsGroup == 3000 && *sc.FSGroup == 2000 && *sc.RunAsNonRoot == true
			}, timeout, interval).Should(BeTrue())

			By("verifying the container resource limits")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "-l", "listjob.batchops.io/name=secure-processor", "-o", "json")
				output, _ := utils.Run(cmd)

				var podList corev1.PodList
				if err := json.Unmarshal([]byte(output), &podList); err != nil || len(podList.Items) == 0 {
					return false
				}

				// Find main container
				var container corev1.Container
				found := false
				for _, c := range podList.Items[0].Spec.Containers {
					if c.Name == "main" {
						container = c
						found = true
						break
					}
				}
				if !found {
					return false
				}

				limits := container.Resources.Limits
				requests := container.Resources.Requests

				return (limits.Cpu().MilliValue() == 2000) && limits.Memory().String() == "512Mi" &&
					(requests.Cpu().MilliValue() == 500) && requests.Memory().String() == "256Mi"
			}, timeout, interval).Should(BeTrue())

			By("verifying the service account name")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "-l", "listjob.batchops.io/name=secure-processor", "-o", "jsonpath={.items[0].spec.serviceAccountName}")
				output, _ := utils.Run(cmd)
				return output
			}, timeout, interval).Should(Equal("parallax-job-runner"))
		})
	})
})
