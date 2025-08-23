# Security Policy

## Supported Versions

We actively support the following versions with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability in Parallax, please report it privately.

### How to Report

1. **Do NOT open a public issue** for security vulnerabilities
2. Email security concerns to: matanryngler@gmail.com 
3. Include as much detail as possible:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

### What to Expect

- **Acknowledgment**: We'll acknowledge receipt within 2 business days
- **Initial Assessment**: We'll provide an initial assessment within 5 business days
- **Updates**: We'll keep you informed of progress every 5 business days
- **Resolution**: We aim to resolve critical issues within 30 days

### Security Measures

Parallax implements several security best practices:

- **Container Security**: All container images are scanned with Trivy
- **Code Scanning**: Static analysis with gosec for Go code
- **Image Signing**: Container images are signed with cosign
- **RBAC**: Minimal required permissions for Kubernetes operations
- **Secrets Management**: Secure handling of API keys and database credentials
- **TLS**: Secure metrics endpoints with proper certificate management

### Security Scanning

Our CI/CD pipeline includes:
- **SAST** (Static Application Security Testing) with gosec
- **Container scanning** with Trivy for known vulnerabilities
- **Dependency scanning** for Go modules
- **SARIF reporting** to GitHub Security tab

### Responsible Disclosure

We follow responsible disclosure practices:
- Security issues are addressed before public disclosure
- We coordinate with reporters on disclosure timing
- Credit is given to security researchers (with permission)

## Security Contact

For security-related questions or concerns:
- Security Email: matanryngler@gmail.com
- Response Time: 2 business days
- PGP Key: [Optional - provide if available]

---

Thank you for helping keep Parallax and our community safe!