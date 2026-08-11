package http

import (
	"errors"
	"net/http"

	"github.com/estudosdevops/sample-api/internal/domain"
	"github.com/estudosdevops/sample-api/internal/usecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	addrUC *usecase.AddressUseCase
}

func NewHandler(auc *usecase.AddressUseCase) *Handler {
	return &Handler{addrUC: auc}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/address/:cep", h.getAddress)
}

func (h *Handler) getAddress(c *gin.Context) {
	cep := c.Param("cep")
	addr, err := h.addrUC.GetByCEP(c.Request.Context(), cep)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, addr)
}
