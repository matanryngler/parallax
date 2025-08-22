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
		// Clean up any helm releases
		cmd := exec.Command("helm", "uninstall", "parallax-test", "-n", helmTestNamespace)
		_, _ = utils.Run(cmd)
		cmd = exec.Command("helm", "uninstall", "parallax-crds-test", "-n", helmTestNamespace)
		_, _ = utils.Run(cmd)

		// Clean up namespace
		cmd = exec.Command("kubectl", "delete", "ns", helmTestNamespace, "--ignore-not-found=true")
		_, _ = utils.Run(cmd)
	})

	SetDefaultEventuallyTimeout(5 * time.Minute)
	SetDefaultEventuallyPollingInterval(10 * time.Second)

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
				"--timeout=120s")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("installing parallax operator chart")
			cmd = exec.Command("helm", "install", "parallax-test", "./charts/parallax",
				"-n", helmTestNamespace,
				"--set", "image.tag=e2e-test",
				"--wait",
				"--timeout=300s")
			output, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("STATUS: deployed"))

			By("verifying CRDs are installed")
			cmd = exec.Command("kubectl", "get", "crd", "listsources.batchops.io")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "get", "crd", "listjobs.batchops.io")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "get", "crd", "listcronjobs.batchops.io")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying operator deployment is ready")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "deployment", "parallax-controller-manager",
					"-n", helmTestNamespace, "-o", "jsonpath={.status.readyReplicas}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("1"))
			}).Should(Succeed())

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
				"--timeout=120s")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("installing parallax chart without CRDs")
			cmd = exec.Command("helm", "install", "parallax-test", "./charts/parallax",
				"-n", helmTestNamespace,
				"--set", "image.tag=e2e-test",
				"--wait",
				"--timeout=300s")
			output, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("STATUS: deployed"))

			By("verifying operator works with separately installed CRDs")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "deployment", "parallax-controller-manager",
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
			cmd = exec.Command("helm", "install", "parallax-test", "./charts/parallax",
				"-n", helmTestNamespace,
				"--set", "image.tag=e2e-test",
				"--set", "resources.limits.cpu=500m",
				"--set", "resources.limits.memory=256Mi",
				"--set", "resources.requests.cpu=100m",
				"--set", "resources.requests.memory=128Mi",
				"--set", "replicaCount=1",
				"--wait",
				"--timeout=300s")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying custom resource configuration")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "deployment", "parallax-controller-manager",
					"-n", helmTestNamespace, "-o", "jsonpath={.spec.template.spec.containers[0].resources.limits.cpu}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("500m"))
			}).Should(Succeed())

			By("verifying custom memory configuration")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "deployment", "parallax-controller-manager",
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
			cmd = exec.Command("helm", "install", "parallax-test", "./charts/parallax",
				"-n", helmTestNamespace,
				"--set", "image.tag=e2e-test",
				"--set", "serviceAccount.name=custom-parallax-sa",
				"--wait",
				"--timeout=300s")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying custom service account is used")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "deployment", "parallax-controller-manager",
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

			By("installing basic configuration")
			cmd = exec.Command("helm", "install", "parallax-test", "./charts/parallax",
				"-n", helmTestNamespace,
				"--set", "image.tag=e2e-test",
				"--wait",
				"--timeout=300s")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying initial installation")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "deployment", "parallax-controller-manager",
					"-n", helmTestNamespace, "-o", "jsonpath={.status.readyReplicas}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("1"))
			}).Should(Succeed())

			By("upgrading with resource limits")
			cmd = exec.Command("helm", "upgrade", "parallax-test", "./charts/parallax",
				"-n", helmTestNamespace,
				"--set", "image.tag=e2e-test",
				"--set", "resources.limits.cpu=800m",
				"--set", "resources.limits.memory=512Mi",
				"--wait",
				"--timeout=300s")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying upgrade applied resource limits")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "deployment", "parallax-controller-manager",
					"-n", helmTestNamespace, "-o", "jsonpath={.spec.template.spec.containers[0].resources.limits.cpu}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("800m"))
			}).Should(Succeed())

			By("verifying operator still works after upgrade with simple test")
			applyTestManifest("listsource-upgrade.yaml", helmTestNamespace)

			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "configmap", "upgrade-test-source", "-n", helmTestNamespace, "-o", "jsonpath={.data.items}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("upgrade-item"))
			}).Should(Succeed())
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

// testBasicFunctionality runs comprehensive basic functionality tests against a Helm-deployed operator
func testBasicFunctionality(namespace string) {
	By("testing static ListSource creation and ConfigMap generation")
	applyTestManifest("listsource-static.yaml", namespace)

	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "configmap", "static-test", "-n", namespace, "-o", "jsonpath={.data.items}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(ContainSubstring("item-1"))
		g.Expect(output).To(ContainSubstring("item-2"))
		g.Expect(output).To(ContainSubstring("item-3"))
	}).Should(Succeed())

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
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(Equal("2"))
	}).Should(Succeed())

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
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(Equal("3"))
	}).Should(Succeed())

	By("testing ListCronJob creation")
	applyTestManifest("listcronjob-static.yaml", namespace)

	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "cronjob", "-l", "listcronjob=static-cronjob-test", "-n", namespace, "-o", "jsonpath={.items[0].spec.schedule}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(Equal("0 */6 * * *"))
	}).Should(Succeed())

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
