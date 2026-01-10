# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x     | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

If you discover a security vulnerability in x402-cli, please report it privately:

1. **GitHub Security Advisories** (Preferred): Go to [Security Advisories](https://github.com/port402/x402-cli/security/advisories/new) and create a new advisory.

2. **Email**: Send details to security@port402.com

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Any suggested fixes (optional)

### Response Timeline

- **Acknowledgment**: Within 48 hours
- **Initial Assessment**: Within 7 days
- **Resolution**: Depends on severity, typically 30-90 days

### What to Expect

1. We will acknowledge your report within 48 hours
2. We will investigate and keep you informed of progress
3. We will credit you in the release notes (unless you prefer anonymity)
4. We will coordinate disclosure timing with you

## Security Considerations

x402-cli handles sensitive data including:

- Private keys (via `--keystore` or `--wallet` flags)
- Payment signatures (EIP-712 typed data)

### Best Practices for Users

- Never share your private key or keystore password
- Use dedicated test wallets for testing
- Keep your keystore files secure with appropriate file permissions
- Review payment amounts before confirming (`--dry-run` flag)
- Use `--max-amount` flag to set spending limits

## Out of Scope

The following are not considered vulnerabilities:

- Issues in third-party dependencies (report to upstream)
- Social engineering attacks
- Physical attacks requiring local access
- Denial of service via excessive API calls
