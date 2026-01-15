package miniapp

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	customerrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/customer"
	customersvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/customer"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type CustomerHandler struct {
	deps *app.Deps
	svc  *customersvc.LocalAuthService
}

func NewCustomerHandler(deps *app.Deps) *CustomerHandler {
	var svc *customersvc.LocalAuthService
	if deps != nil && deps.DB != nil {
		svc = customersvc.NewLocalAuthService(deps.Config, customerrepo.NewRepository(deps.DB))
	}
	return &CustomerHandler{deps: deps, svc: svc}
}

type registerRequest struct {
	TenantUUID string         `json:"tenant_uuid"`
	Email      string         `json:"email"`
	Phone      string         `json:"phone"`
	Password   string         `json:"password" binding:"required"`
	Metadata   map[string]any `json:"metadata"`
}

type loginRequest struct {
	TenantUUID string `json:"tenant_uuid"`
	Login      string `json:"login"`
	// Identifier is a backward-compatible alias of "login" (email or phone).
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

func (h *CustomerHandler) Register(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseInternalError(c, errors.New("customer service unavailable"))
		return
	}

	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid request payload")
		return
	}

	tenantUUID, ok, err := resolveTenantFromHeaderOrBodyOptional(c, req.TenantUUID)
	if err != nil {
		contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeTenantMismatch, err.Error())
		return
	}
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseBadRequest(c, "tenant_uuid is required")
		return
	}

	out, err := h.svc.Register(c.Request.Context(), customersvc.RegisterInput{
		TenantUUID: tenantUUID,
		Email:      req.Email,
		Phone:      req.Phone,
		Password:   req.Password,
		Metadata:   req.Metadata,
	})
	if err != nil {
		switch {
		case errors.Is(err, customerrepo.ErrCustomerExists):
			contracts.ResponseError(c, http.StatusConflict, contracts.ErrCodeConflict, "customer already exists")
		case errors.Is(err, customersvc.ErrCustomerMode):
			contracts.ResponseNotFound(c, "customer register is only available in local mode")
		default:
			contracts.ResponseBadRequest(c, err.Error())
		}
		return
	}

	contracts.ResponseSuccess(c, out)
}

func (h *CustomerHandler) Login(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseInternalError(c, errors.New("customer service unavailable"))
		return
	}

	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid request payload")
		return
	}

	login := strings.TrimSpace(req.Login)
	if login == "" {
		login = strings.TrimSpace(req.Identifier)
	}
	password := req.Password
	if login == "" || strings.TrimSpace(password) == "" {
		contracts.ResponseBadRequest(c, "invalid request payload")
		return
	}

	tenantUUID, _, err := resolveTenantFromHeaderOrBodyOptional(c, req.TenantUUID)
	if err != nil {
		contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeTenantMismatch, err.Error())
		return
	}

	out, err := h.svc.Login(c.Request.Context(), customersvc.LoginInput{
		TenantUUID: tenantUUID,
		Login:      login,
		Password:   password,
	})
	if err != nil {
		var selectionErr *customersvc.TenantSelectionRequiredError
		if errors.As(err, &selectionErr) {
			tenants := make([]map[string]string, 0, len(selectionErr.Tenants))
			for _, t := range selectionErr.Tenants {
				tt := strings.TrimSpace(t)
				if tt == "" {
					continue
				}
				tenants = append(tenants, map[string]string{"tenant_uuid": tt})
			}
			contracts.ResponseErrorWithDetails(c, http.StatusConflict, contracts.ErrCodeTenantSelectionRequired, "tenant selection required", gin.H{
				"tenants": tenants,
			})
			return
		}
		switch {
		case errors.Is(err, customersvc.ErrInvalidCredentials):
			contracts.ResponseUnauthorized(c, "invalid credentials")
		case errors.Is(err, customersvc.ErrCustomerDisabled):
			contracts.ResponseError(c, http.StatusLocked, "CUSTOMER_DISABLED", "customer account disabled")
		case errors.Is(err, customersvc.ErrCustomerDeleted):
			contracts.ResponseUnauthorized(c, "invalid credentials")
		case errors.Is(err, customersvc.ErrCustomerMode):
			contracts.ResponseNotFound(c, "customer login is only available in local mode")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}

	contracts.ResponseSuccess(c, out)
}

func resolveTenantFromHeaderOrBodyOptional(c *gin.Context, bodyTenantUUID string) (string, bool, error) {
	headerTenantUUID, ok := httpmw.TenantUUIDString(c)
	bodyTenantUUID = strings.ToLower(strings.TrimSpace(bodyTenantUUID))
	if ok && strings.TrimSpace(headerTenantUUID) != "" {
		headerTenantUUID = strings.ToLower(strings.TrimSpace(headerTenantUUID))
		if bodyTenantUUID == "" {
			return headerTenantUUID, true, nil
		}
		if bodyTenantUUID != headerTenantUUID {
			return "", false, errors.New("tenant mismatch")
		}
		return headerTenantUUID, true, nil
	}

	if bodyTenantUUID == "" {
		return "", false, nil
	}
	return bodyTenantUUID, true, nil
}
