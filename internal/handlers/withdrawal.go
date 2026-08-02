package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"github.com/internal/models"
	"github.com/internal/service"
)

type WithdrawalHandler struct {
	Service *service.WithdrawalService
}

func NewWithdrawalHandler(svc *service.WithdrawalService) *WithdrawalHandler {
	return &WithdrawalHandler{Service: svc}
}

// POST /withdrawals
func (h *WithdrawalHandler) RequestWithdrawal(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		service.AbortWithError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req models.CreateWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Convert amount and fee to numeric
	amountDec, err := decimal.NewFromString(req.Amount)
	if err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid amount")
		return
	}
	feeDec, err := decimal.NewFromString(req.Fee)
	if err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid fee")
		return
	}

	amountNumeric, _ := service.StringToNumeric(amountDec.String())
	feeNumeric, _ := service.StringToNumeric(feeDec.String())
	totalDebitDec := amountDec.Add(feeDec)
	totalDebitNumeric, _ := service.StringToNumeric(totalDebitDec.String())

	withdrawal, err := h.Service.RequestWithdrawal(
		c.Request.Context(),
		userID,
		req,
		amountNumeric,
		feeNumeric,
		totalDebitNumeric,
	)
	if err != nil {
		if err == service.ErrInsufficientBalance {
			service.AbortWithError(c, http.StatusBadRequest, "insufficient balance")
			return
		}
		service.AbortWithError(c, http.StatusInternalServerError, "failed to create withdrawal")
		return
	}

	c.JSON(http.StatusCreated, withdrawal)
}
