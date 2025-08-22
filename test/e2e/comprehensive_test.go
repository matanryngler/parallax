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
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/matanryngler/parallax/test/utils"
)

const comprehensiveTestNamespace = "parallax-comprehensive-test"

var _ = Describe("Comprehensive Parallax E2E Tests", Ordered, func() {
	BeforeAll(func() {
		By("creating comprehensive test namespace")
		cmd := exec.Command("kubectl", "create", "ns", comprehensiveTestNamespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		By("creating test secrets for authentication")
		createTestSecrets()

		By("starting test infrastructure (API server and PostgreSQL)")
		startTestInfrastructure()
	})

	AfterAll(func() {
		By("stopping test infrastructure")
		stopTestInfrastructure()

		By("cleaning up comprehensive test resources")
		cmd := exec.Command("kubectl", "delete", "ns", comprehensiveTestNamespace)
		_, _ = utils.Run(cmd)
	})

	AfterEach(func() {
		By("cleaning up test resources in namespace")
		cmd := exec.Command("kubectl", "delete", "listjobs,listsources,listcronjobs,jobs,cronjobs", "--all", "-n", comprehensiveTestNamespace)
		_, _ = utils.Run(cmd)
	})

	SetDefaultEventuallyTimeout(10 * time.Minute)
	SetDefaultEventuallyPollingInterval(10 * time.Second)

	Context("API ListSource Tests", func() {
		It("should process simple JSON array from API", func() {
			By("creating API ListSource with simple array")
			apiListSourceYAML := fmt.Sprintf(`
apiVersion: batchops.io/v1alpha1
kind: ListSource
metadata:
  name: api-simple
  namespace: %s
spec:
  type: api
  api:
    url: "http://api-server:8080/simple-array"
    jsonPath: "$[*]"
  intervalSeconds: 30
`, comprehensiveTestNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(apiListSourceYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying API ListSource processes data correctly")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "configmap", "api-simple", "-n", comprehensiveTestNamespace, "-o", "jsonpath={.data.items}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("api-item-1"))
				g.Expect(output).To(ContainSubstring("api-item-2"))
				g.Expect(output).To(ContainSubstring("api-item-3"))
			}).Should(Succeed())
		})

		It("should process complex JSON with JSONPath", func() {
			By("creating API ListSource with complex JSONPath")
			complexAPIYAML := fmt.Sprintf(`
apiVersion: batchops.io/v1alpha1
kind: ListSource
metadata:
  name: api-complex
  namespace: %s
spec:
  type: api
  api:
    url: "http://api-server:8080/complex-json"
    jsonPath: "$.data[*].name"
  intervalSeconds: 30
`, comprehensiveTestNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(complexAPIYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying complex JSONPath extraction works")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "configmap", "api-complex", "-n", comprehensiveTestNamespace, "-o", "jsonpath={.data.items}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("alice"))
				g.Expect(output).To(ContainSubstring("bob"))
				g.Expect(output).To(ContainSubstring("charlie"))
			}).Should(Succeed())
		})

		It("should handle basic authentication", func() {
			By("creating API ListSource with basic auth")
			basicAuthYAML := fmt.Sprintf(`
apiVersion: batchops.io/v1alpha1
kind: ListSource
metadata:
  name: api-basic-auth
  namespace: %s
spec:
  type: api
  api:
    url: "http://api-server:8080/auth/basic"
    jsonPath: "$[*]"
    auth:
      type: basic
      secretRef:
        name: basic-auth-secret
        key: credentials
      usernameKey: username
      passwordKey: password
  intervalSeconds: 30
`, comprehensiveTestNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(basicAuthYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying basic auth works")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "configmap", "api-basic-auth", "-n", comprehensiveTestNamespace, "-o", "jsonpath={.data.items}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("auth-basic-1"))
				g.Expect(output).To(ContainSubstring("auth-basic-2"))
			}).Should(Succeed())
		})

		It("should handle bearer token authentication", func() {
			By("creating API ListSource with bearer token")
			bearerAuthYAML := fmt.Sprintf(`
apiVersion: batchops.io/v1alpha1
kind: ListSource
metadata:
  name: api-bearer-auth
  namespace: %s
spec:
  type: api
  api:
    url: "http://api-server:8080/auth/bearer"
    jsonPath: "$[*]"
    headers:
      Authorization: "Bearer test-token-123"
  intervalSeconds: 30
`, comprehensiveTestNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(bearerAuthYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying bearer token auth works")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "configmap", "api-bearer-auth", "-n", comprehensiveTestNamespace, "-o", "jsonpath={.data.items}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("auth-bearer-1"))
				g.Expect(output).To(ContainSubstring("auth-bearer-2"))
			}).Should(Succeed())
		})
	})

	Context("PostgreSQL ListSource Tests", func() {
		It("should process simple SQL query", func() {
			By("creating PostgreSQL ListSource with simple query")
			pgSimpleYAML := fmt.Sprintf(`
apiVersion: batchops.io/v1alpha1
kind: ListSource
metadata:
  name: pg-simple
  namespace: %s
spec:
  type: postgresql
  postgres:
    connectionString: "host=%s port=5432 dbname=testdb user=testuser password=testpass sslmode=disable"
    query: "SELECT name FROM users WHERE active = true ORDER BY name"
  intervalSeconds: 30
`, comprehensiveTestNamespace, getPostgreSQLHost())

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(pgSimpleYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying PostgreSQL query works")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "configmap", "pg-simple", "-n", comprehensiveTestNamespace, "-o", "jsonpath={.data.items}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Alice Johnson"))
				g.Expect(output).To(ContainSubstring("Bob Smith"))
				g.Expect(output).To(ContainSubstring("Diana Prince"))
				// Should not contain Charlie Brown (inactive user)
				g.Expect(output).NotTo(ContainSubstring("Charlie Brown"))
			}).Should(Succeed())
		})

		It("should process complex SQL query with joins", func() {
			By("creating PostgreSQL ListSource with complex query")
			pgComplexYAML := fmt.Sprintf(`
apiVersion: batchops.io/v1alpha1
kind: ListSource
metadata:
  name: pg-complex
  namespace: %s
spec:
  type: postgresql
  postgres:
    connectionString: "host=%s port=5432 dbname=testdb user=testuser password=testpass sslmode=disable"
    query: "SELECT t.title FROM tasks t JOIN users u ON t.user_id = u.id WHERE u.active = true AND t.status = 'pending' ORDER BY t.priority DESC"
  intervalSeconds: 30
`, comprehensiveTestNamespace, getPostgreSQLHost())

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(pgComplexYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying complex PostgreSQL query works")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "configmap", "pg-complex", "-n", comprehensiveTestNamespace, "-o", "jsonpath={.data.items}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Fix authentication bug"))
				g.Expect(output).To(ContainSubstring("Deploy to staging"))
			}).Should(Succeed())
		})

		It("should handle PostgreSQL authentication with secrets", func() {
			By("creating PostgreSQL ListSource with secret auth")
			pgAuthYAML := fmt.Sprintf(`
apiVersion: batchops.io/v1alpha1
kind: ListSource
metadata:
  name: pg-secret-auth
  namespace: %s
spec:
  type: postgresql
  postgres:
    connectionString: "host=%s port=5432 dbname=testdb sslmode=disable"
    query: "SELECT email FROM users WHERE active = true LIMIT 3"
    auth:
      secretRef:
        name: postgres-secret
        key: password
      passwordKey: password
  intervalSeconds: 30
`, comprehensiveTestNamespace, getPostgreSQLHost())

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(pgAuthYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying PostgreSQL secret auth works")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "configmap", "pg-secret-auth", "-n", comprehensiveTestNamespace, "-o", "jsonpath={.data.items}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("@example.com"))
			}).Should(Succeed())
		})
	})

	Context("Integration Tests", func() {
		It("should create ListJobs from API ListSource", func() {
			By("creating ListJob that references API ListSource")
			listJobYAML := fmt.Sprintf(`
apiVersion: batchops.io/v1alpha1
kind: ListJob
metadata:
  name: api-processor
  namespace: %s
spec:
  listSourceRef: api-simple
  parallelism: 2
  template:
    image: busybox:latest
    command: ["/bin/sh", "-c", "echo Processing API item: $ITEM && sleep 2"]
    envName: ITEM
    resources:
      requests:
        cpu: "50m"
        memory: "32Mi"
`, comprehensiveTestNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(listJobYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying Job is created from API data")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "job", "-l", "listjob=api-processor", "-n", comprehensiveTestNamespace, "-o", "jsonpath={.items[0].spec.completions}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("3")) // Should have 3 completions from API data
			}).Should(Succeed())
		})

		It("should create ListCronJob from PostgreSQL ListSource", func() {
			By("creating ListCronJob that references PostgreSQL ListSource")
			cronJobYAML := fmt.Sprintf(`
apiVersion: batchops.io/v1alpha1
kind: ListCronJob
metadata:
  name: pg-scheduled-processor
  namespace: %s
spec:
  schedule: "0 */6 * * *"  # Every 6 hours
  listSourceRef: pg-simple
  parallelism: 1
  template:
    image: busybox:latest
    command: ["/bin/sh", "-c", "echo Processing DB user: $ITEM"]
    envName: ITEM
    resources:
      requests:
        cpu: "50m"
        memory: "32Mi"
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1
`, comprehensiveTestNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(cronJobYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying CronJob is created from PostgreSQL data")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "cronjob", "-l", "listcronjob=pg-scheduled-processor", "-n", comprehensiveTestNamespace, "-o", "jsonpath={.items[0].spec.schedule}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("0 */6 * * *"))
			}).Should(Succeed())
		})
	})
})

func createTestSecrets() {
	// Create basic auth secret
	basicAuthCmd := exec.Command("kubectl", "create", "secret", "generic", "basic-auth-secret",
		"--from-literal=username=testuser",
		"--from-literal=password=testpass",
		"-n", comprehensiveTestNamespace)
	_, _ = utils.Run(basicAuthCmd)

	// Create PostgreSQL secret
	pgSecretCmd := exec.Command("kubectl", "create", "secret", "generic", "postgres-secret",
		"--from-literal=password=testpass",
		"-n", comprehensiveTestNamespace)
	_, _ = utils.Run(pgSecretCmd)
}

func startTestInfrastructure() {
	// Check if running in CI environment (has postgresql service)
	if isRunningInCI() {
		By("using CI-provided test infrastructure")
		
		By("waiting for API server to be healthy")
		Eventually(func(g Gomega) {
			cmd := exec.Command("curl", "-f", "http://localhost:8080/health")
			_, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
		}, 2*time.Minute, 5*time.Second).Should(Succeed())

		By("waiting for PostgreSQL to be ready")
		Eventually(func(g Gomega) {
			cmd := exec.Command("pg_isready", "-h", "localhost", "-p", "5432", "-U", "testuser", "-d", "testdb")
			_, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
		}, 2*time.Minute, 5*time.Second).Should(Succeed())
	} else {
		By("starting local test infrastructure with docker-compose")
		cmd := exec.Command("docker-compose", "-f", "test/e2e/testdata/docker-compose.yml", "up", "-d")
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for API server to be healthy")
		Eventually(func(g Gomega) {
			cmd := exec.Command("curl", "-f", "http://localhost:8080/health")
			_, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
		}, 2*time.Minute, 5*time.Second).Should(Succeed())

		By("waiting for PostgreSQL to be ready")
		Eventually(func(g Gomega) {
			cmd := exec.Command("docker", "exec", "-i", "postgres", "pg_isready", "-U", "testuser", "-d", "testdb")
			_, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
		}, 2*time.Minute, 5*time.Second).Should(Succeed())
	}
}

func stopTestInfrastructure() {
	if !isRunningInCI() {
		By("stopping test infrastructure")
		cmd := exec.Command("docker-compose", "-f", "test/e2e/testdata/docker-compose.yml", "down", "-v")
		_, _ = utils.Run(cmd)
	}
	// In CI, infrastructure is managed by the workflow
}

func isRunningInCI() bool {
	// Check for GitHub Actions environment
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

func getPostgreSQLHost() string {
	if isRunningInCI() {
		return "localhost"
	}
	return "postgres"
}
