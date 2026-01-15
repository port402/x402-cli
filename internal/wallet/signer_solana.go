package wallet

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/port402/x402-cli/internal/x402"
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
// The transaction is signed by the payer but requires the fee payer's signature
// to be fully valid (facilitator adds this).
func (s *SolanaSigner) Sign(params SignParams) (*SignResult, error) {
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

	sourceATA, _, err := solana.FindAssociatedTokenAddress(payerPubkey, mintPubkey)
	if err != nil {
		return nil, fmt.Errorf("failed to find source ATA: %w", err)
	}

	destATA, _, err := solana.FindAssociatedTokenAddress(recipientPubkey, mintPubkey)
	if err != nil {
		return nil, fmt.Errorf("failed to find destination ATA: %w", err)
	}

	// Build instructions: create destination ATA (if needed) and transfer tokens
	instructions := []solana.Instruction{
		associatedtokenaccount.NewCreateInstruction(
			feePayerPubkey, // fee payer pays for account creation
			recipientPubkey,
			mintPubkey,
		).Build(),
		token.NewTransferInstruction(
			amount,
			sourceATA,
			destATA,
			payerPubkey,
			nil, // no multisig
		).Build(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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
