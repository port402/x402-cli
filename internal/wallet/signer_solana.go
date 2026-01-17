package wallet

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/port402/x402-cli/internal/x402"
)

const (
	// defaultComputeUnitLimit is a reasonable limit for SPL token transfers.
	// The spec doesn't mandate a specific value, but transfers typically need ~50k units.
	defaultComputeUnitLimit uint32 = 200_000

	// defaultComputeUnitPrice in microLamports (1 microLamport = 0.000001 lamports).
	// The spec requires ≤5 lamports/unit. We use 1 microLamport for low priority.
	// For higher priority, facilitators can adjust this.
	defaultComputeUnitPrice uint64 = 1
)

// SolanaSigner implements the Signer interface for Solana payments.
// It creates partially-signed SPL token transfer transactions.
type SolanaSigner struct {
	privateKey solana.PrivateKey
	rpcClient  *rpc.Client
}

// NewSolanaSigner creates a new Solana signer from a private key.
// The rpcURL is used to fetch the recent blockhash for transaction construction.
func NewSolanaSigner(privateKey solana.PrivateKey, rpcURL string) *SolanaSigner {
	return &SolanaSigner{
		privateKey: privateKey,
		rpcClient:  rpc.New(rpcURL),
	}
}

// Sign creates a partially-signed Solana transaction for SPL token transfer.
// The transaction follows the x402 SVM spec with exactly 3 core instructions:
//  1. ComputeBudget::SetComputeUnitLimit
//  2. ComputeBudget::SetComputeUnitPrice
//  3. Token::TransferChecked
//
// If the destination ATA doesn't exist, a CreateAssociatedTokenAccount instruction
// is prepended (making 4 instructions total).
//
// The transaction is signed by the token owner but requires the fee payer's signature
// to be fully valid (facilitator adds this).
func (s *SolanaSigner) Sign(params SignParams) (*SignResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	payerPubkey := s.privateKey.PublicKey()

	recipientPubkey, err := solana.PublicKeyFromBase58(params.To)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient address: %w", err)
	}

	mintPubkey, err := solana.PublicKeyFromBase58(params.TokenAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid token mint address: %w", err)
	}

	feePayerPubkey, err := solana.PublicKeyFromBase58(params.FeePayer)
	if err != nil {
		return nil, fmt.Errorf("invalid fee payer address: %w", err)
	}

	amount, err := strconv.ParseUint(params.Value, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	// Fetch token decimals from mint account (required for TransferChecked)
	decimals, err := s.getTokenDecimals(ctx, mintPubkey)
	if err != nil {
		return nil, fmt.Errorf("failed to get token decimals: %w", err)
	}

	sourceATA, _, err := solana.FindAssociatedTokenAddress(payerPubkey, mintPubkey)
	if err != nil {
		return nil, fmt.Errorf("failed to find source ATA: %w", err)
	}

	destATA, _, err := solana.FindAssociatedTokenAddress(recipientPubkey, mintPubkey)
	if err != nil {
		return nil, fmt.Errorf("failed to find destination ATA: %w", err)
	}

	// Check if destination ATA exists
	destATAExists, err := s.accountExists(ctx, destATA)
	if err != nil {
		return nil, fmt.Errorf("failed to check destination ATA: %w", err)
	}

	// Build instructions per x402 SVM spec:
	// 1. ComputeBudget::SetComputeUnitLimit
	// 2. ComputeBudget::SetComputeUnitPrice
	// 3. Token::TransferChecked
	// (Optional: CreateAssociatedTokenAccount if dest ATA doesn't exist)
	var instructions []solana.Instruction

	// If destination ATA doesn't exist, prepend creation instruction
	if !destATAExists {
		instructions = append(instructions,
			associatedtokenaccount.NewCreateInstruction(
				feePayerPubkey, // fee payer pays for account creation
				recipientPubkey,
				mintPubkey,
			).Build(),
		)
	}

	// ComputeBudget instructions (required by x402 spec)
	instructions = append(instructions,
		computebudget.NewSetComputeUnitLimitInstruction(defaultComputeUnitLimit).Build(),
		computebudget.NewSetComputeUnitPriceInstruction(defaultComputeUnitPrice).Build(),
	)

	// TransferChecked (validates decimals on-chain, required by x402 spec)
	instructions = append(instructions,
		token.NewTransferCheckedInstruction(
			amount,
			decimals,
			sourceATA,
			mintPubkey,
			destATA,
			payerPubkey,
			nil, // no multisig
		).Build(),
	)

	recent, err := s.rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	tx, err := solana.NewTransaction(
		instructions,
		recent.Value.Blockhash,
		solana.TransactionPayer(feePayerPubkey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction: %w", err)
	}

	// Partially sign: only the token owner signs, not the fee payer
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(payerPubkey) {
			return &s.privateKey
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	txBase64, err := tx.ToBase64()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}

	return &SignResult{
		Signature: txBase64,
		Authorization: x402.Authorization{
			From:        params.From,
			To:          params.To,
			Value:       params.Value,
			ValidAfter:  "0",
			ValidBefore: fmt.Sprintf("%d", recent.Value.LastValidBlockHeight),
			Nonce:       base64.StdEncoding.EncodeToString(recent.Value.Blockhash[:]),
		},
		Nonce: recent.Value.Blockhash.String(),
	}, nil
}

// getTokenDecimals fetches the decimal precision for an SPL token mint.
func (s *SolanaSigner) getTokenDecimals(ctx context.Context, mintPubkey solana.PublicKey) (uint8, error) {
	accountInfo, err := s.rpcClient.GetAccountInfo(ctx, mintPubkey)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch mint account: %w", err)
	}

	if accountInfo == nil || accountInfo.Value == nil {
		return 0, fmt.Errorf("mint account not found")
	}

	var mint token.Mint
	if err := mint.Decode(accountInfo.Value.Data.GetBinary()); err != nil {
		return 0, fmt.Errorf("failed to decode mint data: %w", err)
	}

	return mint.Decimals, nil
}

// accountExists checks if an account exists on-chain.
func (s *SolanaSigner) accountExists(ctx context.Context, pubkey solana.PublicKey) (bool, error) {
	accountInfo, err := s.rpcClient.GetAccountInfo(ctx, pubkey)
	if err != nil {
		// RPC returns error for non-existent accounts
		return false, nil //nolint:nilerr // Non-existent account is not an error for our purpose
	}

	return accountInfo != nil && accountInfo.Value != nil, nil
}

// Address returns the base58-encoded public key for this signer.
func (s *SolanaSigner) Address() string {
	return s.privateKey.PublicKey().String()
}

// PrepareSolanaSignParams builds SignParams from a Solana payment requirement.
func PrepareSolanaSignParams(option *x402.PaymentRequirement, fromAddress string) SignParams {
	return SignParams{
		TokenAddress:   option.Asset,
		From:           fromAddress,
		To:             option.PayTo,
		Value:          option.GetAmount(),
		TimeoutSeconds: option.MaxTimeoutSeconds,
		FeePayer:       option.GetExtraString("feePayer"),
	}
}
