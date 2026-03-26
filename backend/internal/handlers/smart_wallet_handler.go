package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ethereum/go-ethereum/common"

	"github.com/web3-lab/backend/internal/services"
	"github.com/web3-lab/backend/pkg/logs"
	"go.uber.org/zap"
)

// SmartWalletHandler handles API integrations covering ERC-4337 Wallet operations.
type SmartWalletHandler struct {
	walletService  *services.SmartWalletService
	bundlerService *services.BundlerService
	// Since we mock verification right now, we assume the requester provides Account ID.
	// In production, this would be extracted securely from an interceptor/middleware
	// verifying the active Kratos Session Cookie and mapping to global Account ID.
}

func NewSmartWalletHandler(ws *services.SmartWalletService, bs *services.BundlerService) *SmartWalletHandler {
	return &SmartWalletHandler{
		walletService:  ws,
		bundlerService: bs,
	}
}

// GetAddress computes the deployed (or soon-to-be-deployed) ERC-4337 Wallet Address for a user.
// GET /api/v1/wallet/address/:account_id
func (h *SmartWalletHandler) GetAddress(c *gin.Context) {
	accountIDParam := c.Param("account_id")
	if accountIDParam == "" {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", "account_id is required"))
		return
	}

	accountID, err := uuid.Parse(accountIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", "invalid account_id format"))
		return
	}

	ownerAddr, _ := h.walletService.GetDeterministicAccount(accountID)
	
	walletAddress, err := h.walletService.DeriveWalletAddress(c.Request.Context(), ownerAddr, accountID)
	if err != nil {
		logs.FromContext(c.Request.Context()).Error("Failed to derive wallet address", zap.Error(err))
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to derive address"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"wallet_address": walletAddress})
}

// ExecuteTransaction acts as the high-level API intent receiver (e.g. "Mint NFT").
// It constructs the UserOp, generates the ZK proof, signs paymaster data, and submits it.
// POST /api/v1/wallet/execute
func (h *SmartWalletHandler) ExecuteTransaction(c *gin.Context) {
	var req struct {
		AccountID    string `json:"account_id" binding:"required"`
		Action       string `json:"action" binding:"required"` // mint, transfer, deploy_contract
		TokenType    string `json:"token_type" binding:"required"` // ERC20, ERC721, ERC1155
		To           string `json:"to"`
		Amount       string `json:"amount"`
		TokenID      string `json:"token_id"`
		TokenAddress string `json:"token_address"`
		Name         string `json:"name"`
		Symbol       string `json:"symbol"`
		Decimals     string `json:"decimals"`
		InitialSupply string `json:"initial_supply"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", err.Error()))
		return
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_REQUEST", "invalid account_id format"))
		return
	}

	// 1. Build UserOperation
	// Using dynamic deterministic EOA slicing to map the Web2 UID -> Web3 Wallet Signer
	ownerAddr, _ := h.walletService.GetDeterministicAccount(accountID)
	senderAddr, err := h.walletService.DeriveWalletAddress(c.Request.Context(), ownerAddr, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("ADDRESS_ERROR", "Failed to get smart wallet address"))
		return
	}

	initCode := "0x"
	code, err := h.bundlerService.GetClient().CodeAt(c.Request.Context(), common.HexToAddress(senderAddr), nil)
	if err == nil && len(code) == 0 {
		initCodeBytes := h.walletService.GetInitCode(ownerAddr, accountID)
		initCode = "0x" + common.Bytes2Hex(initCodeBytes)
	}

	callDataHex, err := h.bundlerService.EncodeExecutionCall(req.Action, req.TokenType, req.To, req.Amount, req.TokenID, senderAddr, req.TokenAddress, req.Name, req.Symbol, req.Decimals, req.InitialSupply)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("PAYLOAD_ERROR", err.Error()))
		return
	}

	userOp, err := h.bundlerService.BuildUserOperation(c.Request.Context(), senderAddr, callDataHex, initCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("BUILD_ERROR", "Failed to build UserOperation"))
		return
	}

	// 2. Sign Paymaster Data natively in backend FIRST! The UserOpHash must cover the paymaster bytes.
	err = h.bundlerService.SignPaymasterData(c.Request.Context(), userOp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("PAYMASTER_ERROR", "Failed signing paymaster data"))
		return
	}

	// 3. Hash UserOp for Prover signing context securely using EIP-4337 v0.7.0 pack standard
	userOpHashBytes, err := h.bundlerService.HashUserOp(c.Request.Context(), userOp)
	dummyUserOpHash := common.Bytes2Hex(userOpHashBytes)
	
	// 4. Generate Authenticated Signature natively wrapping the UserOp
	zkProof, err := h.walletService.GenerateZKProof(c.Request.Context(), accountID, dummyUserOpHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("PROOF_ERROR", "Failed generating proof"))
		return
	}
	userOp.Signature = zkProof

	// 5. Submit heavily-assembled UserOp to the Bundler
	txHash, err := h.bundlerService.SubmitToBundler(c.Request.Context(), userOp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("SUBMIT_ERROR", "Failed to submit to Bundler"))
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"message": "UserOperation submitted successfully via ZK Prover and Paymaster.",
		"mock_proof": zkProof,
		"user_operation": userOp,
		"transaction_hash": txHash,
	})
}
