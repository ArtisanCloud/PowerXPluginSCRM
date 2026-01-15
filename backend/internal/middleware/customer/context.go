package customer

import (
	"context"
	"errors"

	customerdomain "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/customer"
	"github.com/gin-gonic/gin"
)

var ErrCustomerMissing = errors.New("customer context missing")

const ginCustomerContextKey = "customer_ctx"

// SetCustomerContext stores CustomerContext into Gin + request contexts.
func SetCustomerContext(c *gin.Context, cc *customerdomain.CustomerContext) {
	if c == nil || cc == nil {
		return
	}
	c.Set(ginCustomerContextKey, cc)
	if c.Request != nil {
		c.Request = c.Request.WithContext(customerdomain.SetContext(c.Request.Context(), cc))
	}
}

// GetCustomerContext reads CustomerContext from Gin context if present.
func GetCustomerContext(c *gin.Context) (*customerdomain.CustomerContext, bool) {
	if c == nil {
		return nil, false
	}
	if v, ok := c.Get(ginCustomerContextKey); ok && v != nil {
		if cc, ok := v.(*customerdomain.CustomerContext); ok && cc != nil {
			return cc, true
		}
	}
	return nil, false
}

// CustomerContextFromRequest reads CustomerContext from a standard context.
func CustomerContextFromRequest(ctx context.Context) (*customerdomain.CustomerContext, bool) {
	return customerdomain.ContextFrom(ctx)
}

// RequireCustomerContext returns ErrCustomerMissing when absent.
func RequireCustomerContext(c *gin.Context) (*customerdomain.CustomerContext, error) {
	if cc, ok := GetCustomerContext(c); ok {
		return cc, nil
	}
	if c != nil && c.Request != nil {
		if cc, ok := customerdomain.ContextFrom(c.Request.Context()); ok {
			return cc, nil
		}
	}
	return nil, ErrCustomerMissing
}
