// x402 is a CLI tool for testing x402-enabled payment-gated APIs.
//
// The x402 protocol uses HTTP 402 (Payment Required) status codes with
// EIP-3009 gasless token transfers to gate access to resources.
//
// Usage:
//
//	x402 health <url>              Check if endpoint requires payment
//	x402 test <url>                Make a test payment
//	x402 batch-health <file>       Check multiple endpoints
//	x402 version                   Show version info
//
// For more information, visit: https://github.com/port402/x402-cli
package main

import "github.com/port402/x402-cli/internal/commands"

func main() {
	commands.Execute()
}
