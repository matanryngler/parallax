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

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/matanryngler/parallax/test/utils"
)

var _ = Describe("Helm Chart E2E Tests", Ordered, func() {
	const helmTestNamespace = "parallax-helm-test"

	AfterEach(func() {
		By("cleaning up helm releases and namespace")

		// Delete custom resources FIRST while controllers are still running
		// This allows finalizers to be processed properly before controllers shutdown
		By("deleting custom resources while controllers are still active")
		customResourceCommands := [][]string{
			{"kubectl", "delete", "listsources", "--all", "-n", helmTestNamespace, "--timeout=60s"},
			{"kubectl", "delete", "listjobs", "--all", "-n", helmTestNamespace, "--timeout=60s"},
			{"kubectl", "delete", "listcronjobs", "--all", "-n", helmTestNamespace, "--timeout=60s"},
		}

		for _, cmdArgs := range customResourceCommands {
			cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
			output, err := utils.Run(cmd)
			if err != nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "⚠️  Failed to delete resources with %v: %v (output: %s)\n", cmdArgs, err, output)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "✅ Deleted resources: %v\n", cmdArgs)
			}
		}

		// NOW uninstall helm releases (controllers will shutdown gracefully)
		cmd := exec.Command("helm", "uninstall", "parallax-test", "-n", helmTestNamespace, "--timeout=30s")
		_, _ = utils.Run(cmd)
		cmd = exec.Command("helm", "uninstall", "parallax-crds-test", "-n", helmTestNamespace, "--timeout=30s")
		_, _ = utils.Run(cmd)

		// Aggressive namespace cleanup with shorter timeouts
		_, _ = fmt.Fprintf(GinkgoWriter, "🧹 Cleaning up namespace: %s\n", helmTestNamespace)

		// First try: graceful delete with short timeout
		cmd = exec.Command("kubectl", "delete", "ns", helmTestNamespace, "--ignore-not-found=true", "--timeout=30s")
		_, err := utils.Run(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "⚠️  Graceful namespace deletion failed, trying force delete\n")

			// Second try: force delete
			cmd = exec.Command("kubectl", "delete", "ns", helmTestNamespace, "--ignore-not-found=true", "--force", "--grace-period=0", "--timeout=20s")
			_, err = utils.Run(cmd)
			if err != nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "⚠️  Force delete failed, cleaning up finalizers\n")

				// Third try: remove finalizers and force delete
				cmd = exec.Command("kubectl", "patch", "ns", helmTestNamespace, "-p", `{"metadata":{"finalizers":null}}`, "--type=merge", "--ignore-not-found=true")
				_, _ = utils.Run(cmd)
				cmd = exec.Command("kubectl", "delete", "ns", helmTestNamespace, "--ignore-not-found=true", "--force", "--grace-period=0", "--timeout=10s")
				_, _ = utils.Run(cmd)
			}
		}
		_, _ = fmt.Fprintf(GinkgoWriter, "✅ Namespace cleanup completed\n")
	})

	SetDefaultEventuallyTimeout(90 * time.Second)
	SetDefaultEventuallyPollingInterval(3 * time.Second)

	// Add more verbose output for debugging
	BeforeEach(func() {
		GinkgoWriter.Printf("\n🧪 Starting test in namespace: %s\n", helmTestNamespace)
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			GinkgoWriter.Printf("\n❌ Test failed, gathering debug information for namespace: %s\n", helmTestNamespace)
			utils.DebugNamespace(helmTestNamespace)
			utils.GetControllerLogs("parallax-test", helmTestNamespace, 100)
		}
	})

	Context("Fresh Installation", func() {
		It("should install parallax chart with CRDs and verify functionality", func() {
			By("creating test namespace")
			cmd := exec.Command("kubectl", "create", "ns", helmTestNamespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("installing CRDs chart first")
			cmd = exec.Command("helm", "install", "parallax-crds-test", "./charts/parallax-crds",
				"-n", helmTestNamespace,
				"--wait",
				"--timeout=60s")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("installing parallax operator chart")
			args := []string{"install", "parallax-test", "./charts/parallax", "-n", helmTestNamespace}
			args = append(args, getHelmImageSettings()...)
			args = append(args, "--wait", "--timeout=60s")
			cmd = exec.Command("helm", args...)
			output, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("STATUS: deployed"))

			// CRDs already verified in previous step

			// Deployment readiness already verified in previous step

			// Run comprehensive basic functionality tests
			testBasicFunctionality(helmTestNamespace)
		})

		It("should install parallax chart with separate CRDs chart", func() {
			By("creating test namespace")
			cmd := exec.Command("kubectl", "create", "ns", helmTestNamespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("installing CRDs separately")
			cmd = exec.Command("helm", "install", "parallax-crds-test", "./charts/parallax-crds",
				"-n", helmTestNamespace,
				"--wait",
				"--timeout=60s")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("installing parallax chart without CRDs")
			args := []string{"install", "parallax-test", "./charts/parallax", "-n", helmTestNamespace}
			args = append(args, getHelmImageSettings()...)
			args = append(args, "--wait", "--timeout=60s")
			cmd = exec.Command("helm", args...)
			output, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("STATUS: deployed"))

			By("verifying operator works with separately installed CRDs")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "deployment", "parallax-test",
					"-n", helmTestNamespace, "-o", "jsonpath={.status.readyReplicas}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("1"))
			}).Should(Succeed())
		})
	})

	Context("Configuration Options", func() {
		It("should install with custom resource configurations", func() {
			By("creating test namespace")
			cmd := exec.Command("kubectl", "create", "ns", helmTestNamespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("installing with custom resource limits")
			args := []string{"install", "parallax-test", "./charts/parallax", "-n", helmTestNamespace}
			args = append(args, getHelmImageSettings()...)
			args = append(args,
				"--set", "resources.limits.cpu=500m",
				"--set", "resources.limits.memory=256Mi",
				"--set", "resources.requests.cpu=100m",
				"--set", "resources.requests.memory=128Mi",
				"--set", "replicaCount=1",
				"--wait",
				"--timeout=60s")
			cmd = exec.Command("helm", args...)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying custom resource configuration")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "deployment", "parallax-test",
					"-n", helmTestNamespace, "-o", "jsonpath={.spec.template.spec.containers[0].resources.limits.cpu}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("500m"))
			}).Should(Succeed())

			By("verifying custom memory configuration")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "deployment", "parallax-test",
					"-n", helmTestNamespace, "-o", "jsonpath={.spec.template.spec.containers[0].resources.requests.memory}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("128Mi"))
			}).Should(Succeed())
		})

		It("should install with custom service account", func() {
			By("creating test namespace")
			cmd := exec.Command("kubectl", "create", "ns", helmTestNamespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("installing with custom service account name")
			args := []string{"install", "parallax-test", "./charts/parallax", "-n", helmTestNamespace}
			args = append(args, getHelmImageSettings()...)
			args = append(args,
				"--set", "serviceAccount.name=custom-parallax-sa",
				"--wait",
				"--timeout=60s")
			cmd = exec.Command("helm", args...)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying custom service account is used")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "deployment", "parallax-test",
					"-n", helmTestNamespace, "-o", "jsonpath={.spec.template.spec.serviceAccountName}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("custom-parallax-sa"))
			}).Should(Succeed())

			By("verifying service account exists")
			cmd = exec.Command("kubectl", "get", "sa", "custom-parallax-sa", "-n", helmTestNamespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Upgrade Scenarios", func() {
		It("should upgrade from basic to resource-configured installation", func() {
			By("creating test namespace")
			cmd := exec.Command("kubectl", "create", "ns", helmTestNamespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("installing CRDs first")
			cmd = exec.Command("helm", "install", "parallax-crds-test", "./charts/parallax-crds",
				"-n", helmTestNamespace,
				"--wait",
				"--timeout=60s")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying CRDs are established")
			err = utils.WaitForCRD("listsources.batchops.io", 30)
			Expect(err).NotTo(HaveOccurred())
			err = utils.WaitForCRD("listjobs.batchops.io", 30)
			Expect(err).NotTo(HaveOccurred())
			err = utils.WaitForCRD("listcronjobs.batchops.io", 30)
			Expect(err).NotTo(HaveOccurred())

			By("installing basic configuration")
			args := []string{"install", "parallax-test", "./charts/parallax", "-n", helmTestNamespace}
			args = append(args, getHelmImageSettings()...)
			args = append(args, "--wait", "--timeout=60s")
			cmd = exec.Command("helm", args...)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for controller to be ready")
			err = utils.WaitForDeployment("parallax-test", helmTestNamespace, 60)
			if err != nil {
				utils.GetDeploymentStatus("parallax-test", helmTestNamespace)
				utils.GetControllerLogs("parallax-test", helmTestNamespace, 50)
			}
			Expect(err).NotTo(HaveOccurred())

			By("verifying initial installation")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "deployment", "parallax-test",
					"-n", helmTestNamespace, "-o", "jsonpath={.status.readyReplicas}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("1"))
			}).Should(Succeed())

			By("upgrading with resource limits")
			args = []string{"upgrade", "parallax-test", "./charts/parallax", "-n", helmTestNamespace}
			args = append(args, getHelmImageSettings()...)
			args = append(args,
				"--set", "resources.limits.cpu=800m",
				"--set", "resources.limits.memory=512Mi",
				"--wait",
				"--timeout=60s")
			cmd = exec.Command("helm", args...)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for upgraded controller to be ready")
			err = utils.WaitForDeployment("parallax-test", helmTestNamespace, 60)
			if err != nil {
				utils.GetDeploymentStatus("parallax-test", helmTestNamespace)
				utils.GetControllerLogs("parallax-test", helmTestNamespace, 50)
			}
			Expect(err).NotTo(HaveOccurred())

			By("verifying upgrade applied resource limits")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "deployment", "parallax-test",
					"-n", helmTestNamespace, "-o", "jsonpath={.spec.template.spec.containers[0].resources.limits.cpu}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("800m"))
			}).Should(Succeed())

			By("verifying operator still works after upgrade with simple test")
			applyTestManifest("listsource-upgrade.yaml", helmTestNamespace)

			By("waiting for ListSource to be processed by controller")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "listsource", "upgrade-test-source", "-n", helmTestNamespace, "-o", "jsonpath={.status.state}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Ready"))
			}, 120, 5).Should(Succeed())

			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "configmap", "upgrade-test-source", "-n", helmTestNamespace, "-o", "jsonpath={.data.items}")
				output, err := utils.Run(cmd)
				if err != nil {
					// Debug on failure
					utils.DebugNamespace(helmTestNamespace)
					utils.GetControllerLogs("parallax-test", helmTestNamespace, 100)
				}
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("upgrade-item"))
			}, 90, 5).Should(Succeed())
		})
	})

	Context("Chart Validation", func() {
		It("should lint both charts successfully", func() {
			By("linting main parallax chart")
			cmd := exec.Command("helm", "lint", "./charts/parallax")
			output, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("1 chart(s) linted, 0 chart(s) failed"))

			By("linting parallax-crds chart")
			cmd = exec.Command("helm", "lint", "./charts/parallax-crds")
			output, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("1 chart(s) linted, 0 chart(s) failed"))
		})

		It("should template charts without errors", func() {
			By("templating main chart with default values")
			cmd := exec.Command("helm", "template", "test-release", "./charts/parallax")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("templating main chart with custom values")
			cmd = exec.Command("helm", "template", "test-release", "./charts/parallax",
				"--set", "image.tag=custom-tag",
				"--set", "replicaCount=2")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("templating CRDs chart")
			cmd = exec.Command("helm", "template", "test-crds", "./charts/parallax-crds")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

// applyTestManifest applies a test manifest from testdata directory with namespace substitution
func applyTestManifest(filename, namespace string) {
	manifestPath := filepath.Join("test", "e2e", "testdata", filename)
	yamlBytes, err := os.ReadFile(manifestPath)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to read manifest %s", manifestPath))

	// Replace namespace placeholder
	yamlContent := strings.ReplaceAll(string(yamlBytes), "{{NAMESPACE}}", namespace)

	By(fmt.Sprintf("applying test manifest %s to namespace %s", filename, namespace))
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(yamlContent)
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())
}

// getHelmImageSettings returns the correct image settings for Helm installations
func getHelmImageSettings() []string {
	repository := os.Getenv("HELM_IMAGE_REPOSITORY")
	if repository == "" {
		repository = "ghcr.io/matanryngler/parallax"
	}

	tag := os.Getenv("HELM_IMAGE_TAG")
	if tag == "" {
		tag = "e2e-test"
	}

	pullPolicy := os.Getenv("HELM_IMAGE_PULL_POLICY")
	if pullPolicy == "" {
		pullPolicy = "IfNotPresent"
	}

	return []string{
		"--set", fmt.Sprintf("image.repository=%s", repository),
		"--set", fmt.Sprintf("image.tag=%s", tag),
		"--set", fmt.Sprintf("image.pullPolicy=%s", pullPolicy),
	}
}

// testBasicFunctionality runs comprehensive basic functionality tests against a Helm-deployed operator
func testBasicFunctionality(namespace string) {
	By("testing static ListSource creation and ConfigMap generation")
	applyTestManifest("listsource-static.yaml", namespace)

	By("waiting for ListSource to be processed")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "listsource", "static-test", "-n", namespace, "-o", "jsonpath={.status.state}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(Equal("Ready"))
	}, 60, 5).Should(Succeed())

	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "configmap", "static-test", "-n", namespace, "-o", "jsonpath={.data.items}")
		output, err := utils.Run(cmd)
		if err != nil {
			// Debug on failure
			utils.DebugNamespace(namespace)
			utils.GetControllerLogs("parallax-test", namespace, 50)
		}
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(ContainSubstring("item-1"))
		g.Expect(output).To(ContainSubstring("item-2"))
		g.Expect(output).To(ContainSubstring("item-3"))
	}, 120, 10).Should(Succeed())

	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "listsource", "static-test", "-n", namespace, "-o", "jsonpath={.status.itemCount}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(Equal("3"))
	}).Should(Succeed())

	By("testing ListJob creation from static list")
	applyTestManifest("listjob-static.yaml", namespace)

	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "job", "-l", "listjob=static-job-test", "-n", namespace, "-o", "jsonpath={.items[0].spec.completions}")
		output, err := utils.Run(cmd)
		if err != nil {
			// Debug on failure
			utils.DebugNamespace(namespace)
			utils.GetControllerLogs("parallax-test", namespace, 50)
		}
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(Equal("2"))
	}, 120, 10).Should(Succeed())

	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "job", "-l", "listjob=static-job-test", "-n", namespace, "-o", "jsonpath={.items[0].spec.completionMode}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(Equal("Indexed"))
	}).Should(Succeed())

	By("testing ListJob creation from ListSource reference")
	applyTestManifest("listsource-ref.yaml", namespace)

	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "configmap", "ref-source", "-n", namespace)
		_, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
	}).Should(Succeed())

	applyTestManifest("listjob-ref.yaml", namespace)

	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "job", "-l", "listjob=ref-job-test", "-n", namespace, "-o", "jsonpath={.items[0].spec.completions}")
		output, err := utils.Run(cmd)
		if err != nil {
			// Debug on failure
			utils.DebugNamespace(namespace)
			utils.GetControllerLogs("parallax-test", namespace, 50)
		}
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(Equal("3"))
	}, 120, 10).Should(Succeed())

	By("testing ListCronJob creation")
	applyTestManifest("listcronjob-static.yaml", namespace)

	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "cronjob", "-l", "listcronjob=static-cronjob-test", "-n", namespace, "-o", "jsonpath={.items[0].spec.schedule}")
		output, err := utils.Run(cmd)
		if err != nil {
			// Debug on failure
			utils.DebugNamespace(namespace)
			utils.GetControllerLogs("parallax-test", namespace, 50)
		}
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(Equal("0 */6 * * *"))
	}, 120, 10).Should(Succeed())

	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "cronjob", "-l", "listcronjob=static-cronjob-test", "-n", namespace, "-o", "jsonpath={.items[0].spec.jobTemplate.spec.completions}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(Equal("2"))
	}).Should(Succeed())

	By("cleaning up basic functionality test resources")
	cmd := exec.Command("kubectl", "delete", "listjobs,listsources,listcronjobs,jobs,cronjobs", "--all", "-n", namespace)
	_, _ = utils.Run(cmd)
}
