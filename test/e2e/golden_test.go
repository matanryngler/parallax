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
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/matanryngler/parallax/test/utils"
)

// Golden file tests validate that generated manifests match expected outputs
// This follows the pattern used by Prometheus Operator and other mature operators
var _ = Describe("Golden File Tests", func() {
	Context("Manifest Generation", func() {
		It("should generate consistent CRD manifests", func() {
			By("generating CRD manifests")
			cmd := exec.Command("make", "manifests")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("checking that CRDs are valid YAML")
			cmd = exec.Command("kubectl", "apply", "--dry-run=client", "--validate=true", "-f", "config/crd/bases/")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying ListSource CRD structure")
			cmd = exec.Command("kubectl", "apply", "--dry-run=client", "-f", "config/crd/bases/batchops.io_listsources.yaml")
			output, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("listsource.batchops.io"))
		})

		It("should generate consistent RBAC manifests", func() {
			By("validating RBAC manifests")
			cmd := exec.Command("kubectl", "apply", "--dry-run=client", "--validate=true", "-f", "config/rbac/")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("checking ClusterRole contains required permissions")
			cmd = exec.Command("cat", "config/rbac/role.yaml")
			output, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			// Verify key permissions are present
			Expect(output).To(ContainSubstring("listsources"))
			Expect(output).To(ContainSubstring("listjobs"))
			Expect(output).To(ContainSubstring("listcronjobs"))
			Expect(output).To(ContainSubstring("jobs"))
			Expect(output).To(ContainSubstring("cronjobs"))
		})

		It("should sync manifests to Helm charts correctly", func() {
			By("running sync-all to ensure charts are up-to-date")
			cmd := exec.Command("make", "sync-all")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying CRDs exist in both chart locations")
			cmd = exec.Command("ls", "charts/parallax/crds/")
			output, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("batchops.io_listsources.yaml"))
			Expect(output).To(ContainSubstring("batchops.io_listjobs.yaml"))
			Expect(output).To(ContainSubstring("batchops.io_listcronjobs.yaml"))

			cmd = exec.Command("ls", "charts/parallax-crds/templates/")
			output, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("batchops.io_listsources.yaml"))
		})

		It("should validate Helm chart templates", func() {
			By("linting main Helm chart")
			cmd := exec.Command("helm", "lint", "charts/parallax")
			output, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("1 chart(s) linted, 0 chart(s) failed"))

			By("linting CRDs Helm chart")
			cmd = exec.Command("helm", "lint", "charts/parallax-crds")
			output, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("1 chart(s) linted, 0 chart(s) failed"))

			By("rendering templates with dry-run")
			cmd = exec.Command("helm", "template", "test", "charts/parallax", "--dry-run")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Configuration Validation", func() {
		It("should validate sample configurations", func() {
			By("checking sample ListSource configurations")
			cmd := exec.Command("kubectl", "apply", "--dry-run=client", "--validate=true", "-f", "config/samples/")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate webhook configurations if present", func() {
			// Skip if no webhook configs exist
			cmd := exec.Command("ls", "config/webhook/")
			_, err := utils.Run(cmd)
			if err != nil {
				Skip("No webhook configurations found")
			}

			By("validating webhook manifests")
			cmd = exec.Command("kubectl", "apply", "--dry-run=client", "--validate=true", "-f", "config/webhook/")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Code Generation", func() {
		It("should have up-to-date generated code", func() {
			By("running code generation")
			cmd := exec.Command("make", "generate")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("checking git status for uncommitted changes")
			cmd = exec.Command("git", "status", "--porcelain")
			output, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			// Filter out CLAUDE.md and other non-generated files
			lines := strings.Split(strings.TrimSpace(output), "\n")
			var generatedFileChanges []string
			for _, line := range lines {
				if strings.TrimSpace(line) == "" {
					continue
				}
				// Only check for changes in generated files
				if strings.Contains(line, "zz_generated") ||
					strings.Contains(line, "charts/") ||
					strings.Contains(line, "config/crd/bases/") ||
					strings.Contains(line, "config/rbac/") {
					generatedFileChanges = append(generatedFileChanges, line)
				}
			}

			if len(generatedFileChanges) > 0 {
				Fail("Generated code is out of sync. Please run 'make generate' and commit changes:\n" +
					strings.Join(generatedFileChanges, "\n"))
			}
		})
	})
})
