package service

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- UUID ---
var ErrInsufficientBalance = errors.New("insufficient balance")

func UUIDToPGType(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: u,
		Valid: true,
	}
}

func PGTypeToUUID(u pgtype.UUID) uuid.UUID {
	return uuid.UUID(u.Bytes)
}

// PGUUIDToPtr returns nil for a NULL/invalid pgtype.UUID, otherwise a pointer
// to the parsed uuid.UUID — useful for optional foreign keys like object_id.
func PGUUIDToPtr(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes)
	return &id
}

// --- Text / string ---

// StringToPGText always marks the value valid, even for an empty string.
// Use this for required-but-possibly-empty text fields.
func StringToPGText(s string) pgtype.Text {
	return pgtype.Text{
		String: s,
		Valid:  true,
	}
}

// StringToPGTextNullable treats an empty string as SQL NULL — use this for
// genuinely optional fields (e.g. ban_reason, tx_hash) where "" and "not set"
// should be indistinguishable.
func StringToPGTextNullable(s string) pgtype.Text {
	return pgtype.Text{
		String: s,
		Valid:  s != "",
	}
}

func PGTextToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

// PGTextToPtr returns nil for NULL — matches how optional JSON fields
// (e.g. `TxHash *string`) should serialize.
func PGTextToPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

// PtrToPGText is the inverse — nil pointer becomes SQL NULL.
func PtrToPGText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// --- Numeric (money/quantity fields — NUMERIC columns) ---

// StringToNumeric parses a decimal string (e.g. from a JSON request body)
// into pgtype.Numeric. This is the preferred path for user-supplied amounts,
// since it avoids float64 entirely and preserves exact decimal precision.
func StringToNumeric(s string) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.Scan(s); err != nil {
		return pgtype.Numeric{}, fmt.Errorf("invalid decimal value %q: %w", s, err)
	}
	return n, nil
}

// MustStringToNumeric panics on invalid input — only use for hardcoded
// literals you control (e.g. zero-value constants), never user input.
func MustStringToNumeric(s string) pgtype.Numeric {
	n, err := StringToNumeric(s)
	if err != nil {
		panic(err)
	}
	return n
}

// NumericToString converts a pgtype.Numeric back to its decimal string
// representation — safe to send directly in a JSON response (as a string,
// never as a JSON number, to avoid client-side float precision loss).
func NumericToString(n pgtype.Numeric) (string, error) {
	if !n.Valid {
		return "0", nil
	}
	val, err := n.Value()
	if err != nil {
		return "", fmt.Errorf("failed to read numeric value: %w", err)
	}
	switch v := val.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// Float64ToNumeric exists for internal calculations (e.g. computed fees) —
// avoid using this for values that originated as user input or came
// straight from the database, since float64 can't represent all decimals
// exactly. Prefer StringToNumeric for anything precision-sensitive.
func Float64ToNumeric(f float64) (pgtype.Numeric, error) {
	return StringToNumeric(strconv.FormatFloat(f, 'f', -1, 64))
}

// --- Timestamps ---

func TimeToPGTimestamp(t time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: t, Valid: true}
}

// PGTimestampToPtr returns nil for NULL — matches optional JSON fields like
// `FilledAt *time.Time` or `ConfirmedAt *time.Time`.
func PGTimestampToPtr(t pgtype.Timestamp) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// --- Numbers ---

// SafeInt coerces a loosely-typed value (e.g. from decoded JSON, which
// unmarshals all numbers as float64) into an int. Returns 0 for nil or any
// type it doesn't recognize, including unparsable strings.
func SafeInt(value interface{}) int {
	if value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}
// userIDFromContext reads the userID set by AuthMiddleware via c.Set("userID", userUUID).
// ok=false means this handler was reached without going through AuthMiddleware first —
// that's a routing setup bug, not a normal unauthenticated request.
func UserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("userID")
	if !exists {
		return uuid.UUID{}, false
	}
	id, ok := val.(uuid.UUID)
	return id, ok
}

// roleFromContext reads the role set by AuthMiddleware.
func RoleFromContext(c *gin.Context) string {
	val, exists := c.Get("role")
	if !exists {
		return ""
	}
	role, _ := val.(string)
	return role
}

// parseUUIDParam reads a Gin route param and parses it as a UUID,
// e.g. parseUUIDParam(c, "id") for a route like /api/orders/:id
func ParseUUIDParam(c *gin.Context, name string) (uuid.UUID, error) {
	return uuid.Parse(c.Param(name))
}

// writeError sends a consistent { "error": "..." } JSON body.
// Does NOT call c.Abort() — callers must still `return` immediately after,
// same as your existing AuthMiddleware pattern with AbortWithStatusJSON.
func WriteError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// abortWithError is writeError + c.Abort() in one call, for handlers that
// don't need to run any further middleware/handlers after responding.
func AbortWithError(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}

func WriteJSON(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}
func CoinIDToSymbol(coinID string) string {
	mapping := map[string]string{
		"bitcoin":     "BTCUSDT",
		"ethereum":    "ETHUSDT",
		"solana":      "SOLUSDT",
		"binancecoin": "BNBUSDT",
		"ripple":      "XRPUSDT",
	}
	if symbol, ok := mapping[coinID]; ok {
		return symbol
	}
	return coinID + "USDT"
}

// NumericToFloat64 converts pgtype.Numeric to float64
func NumericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid || n.Int == nil {
		return 0
	}
	f := new(big.Float).SetInt(n.Int)
	if n.Exp != 0 {
		exp := new(big.Float).SetFloat64(math.Pow10(int(-n.Exp)))
		f.Mul(f, exp)
	}
	result, _ := f.Float64()
	return result
}