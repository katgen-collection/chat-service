package handlers

import (
	"mikhailjbs/chat-service/internal/domain/contacts"
	"mikhailjbs/chat-service/internal/infra/middleware"
	ws "mikhailjbs/chat-service/internal/infra/websocket"
	usecase "mikhailjbs/chat-service/internal/usecase/contacts"

	"github.com/gofiber/fiber/v2"
)

type ContactRequestHandler interface {
	CreateContactRequest(c *fiber.Ctx) error
	GetContactRequest(c *fiber.Ctx) error
	GetContactRequests(c *fiber.Ctx) error
	UpdateContactRequest(c *fiber.Ctx) error
	DeleteContactRequest(c *fiber.Ctx) error
}

type contactRequestHandler struct {
	createContactRequestUseCase usecase.CreateContactRequestUseCase
	getContactRequestUseCase    usecase.GetContactRequestUseCase
	getContactRequestsUseCase   usecase.GetContactRequestsUseCase
	updateContactRequestUseCase usecase.UpdateContactRequestUseCase
	deleteContactRequestUseCase usecase.DeleteContactRequestUseCase
	wsHub                       *ws.Hub
}

func NewContactRequestHandler(
	createContactRequestUseCase usecase.CreateContactRequestUseCase,
	getContactRequestUseCase usecase.GetContactRequestUseCase,
	getContactRequestsUseCase usecase.GetContactRequestsUseCase,
	updateContactRequestUseCase usecase.UpdateContactRequestUseCase,
	deleteContactRequestUseCase usecase.DeleteContactRequestUseCase,
	wsHub *ws.Hub,
) ContactRequestHandler {
	return &contactRequestHandler{
		createContactRequestUseCase: createContactRequestUseCase,
		getContactRequestUseCase:    getContactRequestUseCase,
		getContactRequestsUseCase:   getContactRequestsUseCase,
		updateContactRequestUseCase: updateContactRequestUseCase,
		deleteContactRequestUseCase: deleteContactRequestUseCase,
		wsHub:                       wsHub,
	}
}

// CreateContactRequest godoc
// @Summary Send contact request
// @Description Sends a new contact request to another user.
// @Tags ContactRequests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body contacts.CreateContactRequest true "Create Contact Request"
// @Success 201 {object} handlers.SuccessResponse{data=contacts.ContactRequest}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 409 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/contact-requests [post]
func (h *contactRequestHandler) CreateContactRequest(c *fiber.Ctx) error {
	claims, ok := middleware.ClaimsFromContext(c)
	if (len(claims.Roles) == 0) || !ok {
		return SendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	userId := claims.UserID

	var req contacts.CreateContactRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Set requester
	req.SenderID = userId

	// Extract token to pass to usecase for resolving display names
	bearerToken := extractTokenFromRequest(c)

	createdContactRequest, err := h.createContactRequestUseCase.Execute(c.Context(), &req, bearerToken)
	if err != nil {
		if err == contacts.ErrDuplicateRequest {
			return SendError(c, fiber.StatusConflict, err.Error())
		}
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Notify the receiver via WebSocket
	_ = h.wsHub.SendToUser(createdContactRequest.ReceiverID, ws.EventContactRequestIncoming, createdContactRequest)

	return SendSuccess(c, fiber.StatusCreated, "Contact request created successfully", createdContactRequest)
}

// GetContactRequest godoc
// @Summary Get contact request details
// @Description Fetches details of a specific contact request.
// @Tags ContactRequests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Request ID"
// @Success 200 {object} handlers.SuccessResponse{data=contacts.ContactRequest}
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 403 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/contact-requests/{id} [get]
func (h *contactRequestHandler) GetContactRequest(c *fiber.Ctx) error {
	claims, ok := middleware.ClaimsFromContext(c)
	if (len(claims.Roles) == 0) || !ok {
		return SendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	userId := claims.UserID

	id := c.Params("id")
	contactRequest, err := h.getContactRequestUseCase.Execute(c.Context(), id)
	if err != nil {
		if err == contacts.ErrContactRequestNotFound {
			return SendError(c, fiber.StatusNotFound, err.Error())
		}
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Ensure the user is either the sender or receiver
	if contactRequest.SenderID != userId && contactRequest.ReceiverID != userId {
		return SendError(c, fiber.StatusForbidden, "Forbidden")
	}

	return SendSuccess(c, fiber.StatusOK, "Contact request retrieved successfully", contactRequest)
}

// GetContactRequests godoc
// @Summary List contact requests
// @Description Retrieves contact requests involving the current user.
// @Tags ContactRequests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status (pending, accepted, rejected)"
// @Param sender_id query string false "Filter by sender ID"
// @Param receiver_id query string false "Filter by receiver ID"
// @Param limit query int false "Pagination limit"
// @Param offset query int false "Pagination offset"
// @Success 200 {object} handlers.SuccessResponse{data=[]contacts.ContactRequest}
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 403 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/contact-requests [get]
func (h *contactRequestHandler) GetContactRequests(c *fiber.Ctx) error {
	claims, ok := middleware.ClaimsFromContext(c)
	if (len(claims.Roles) == 0) || !ok {
		return SendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	userId := claims.UserID
	userRole := claims.Roles[0]

	var queryParams contacts.ContactRequestQueryParams
	if err := c.QueryParser(&queryParams); err != nil {
		return SendError(c, fiber.StatusBadRequest, "Invalid query parameters")
	}

	// Regular users can only see requests where they are sender or receiver.
	// If no filter provided, force both to the current user to fetch all related requests.
	if userRole == "user" {
		if queryParams.SenderID == "" && queryParams.ReceiverID == "" {
			queryParams.SenderID = userId
			queryParams.ReceiverID = userId
		} else if queryParams.SenderID != userId && queryParams.ReceiverID != userId {
			return SendError(c, fiber.StatusForbidden, "Forbidden")
		}
	}

	contactRequests, err := h.getContactRequestsUseCase.Execute(c.Context(), &queryParams)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, "Contact requests retrieved successfully", contactRequests)
}

// UpdateContactRequest godoc
// @Summary Update contact request
// @Description Accept or reject a contact request.
// @Tags ContactRequests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Request ID"
// @Param request body contacts.UpdateContactRequest true "Update Contact Request"
// @Success 200 {object} handlers.SuccessResponse{data=contacts.ContactRequest}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 403 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/contact-requests/{id} [put]
func (h *contactRequestHandler) UpdateContactRequest(c *fiber.Ctx) error {
	id := c.Params("id")
	claims, ok := middleware.ClaimsFromContext(c)
	if (len(claims.Roles) == 0) || !ok {
		return SendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	userId := claims.UserID
	userRole := claims.Roles[0]

	var req contacts.UpdateContactRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Regular users can only update their own contact requests
	if userRole == "user" {
		existingRequest, err := h.getContactRequestUseCase.Execute(c.Context(), id)
		if err != nil {
			return SendError(c, fiber.StatusInternalServerError, err.Error())
		}
		if existingRequest == nil {
			return SendError(c, fiber.StatusNotFound, "Contact request not found")
		}
		if existingRequest.SenderID != userId && existingRequest.ReceiverID != userId {
			return SendError(c, fiber.StatusForbidden, "Forbidden")
		}
	}

	// Extract JWT token from request to forward to user service
	bearerToken := extractTokenFromRequest(c)

	updatedContactRequest, err := h.updateContactRequestUseCase.Execute(c.Context(), id, &req, bearerToken)
	if err != nil {
		if err == contacts.ErrContactRequestNotFound {
			return SendError(c, fiber.StatusNotFound, err.Error())
		}
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Notify the opposite party
	oppositeUser := updatedContactRequest.SenderID
	if userId == updatedContactRequest.SenderID {
		oppositeUser = updatedContactRequest.ReceiverID
	}
	_ = h.wsHub.SendToUser(oppositeUser, ws.EventContactRequestUpdated, updatedContactRequest)

	if updatedContactRequest.Status == "accepted" {
		h.wsHub.NotifyNewContact(updatedContactRequest.SenderID, updatedContactRequest.ReceiverID)
	}

	return SendSuccess(c, fiber.StatusOK, "Contact request updated successfully", updatedContactRequest)
}

// DeleteContactRequest godoc
// @Summary Delete contact request
// @Description Cancels or removes a contact request.
// @Tags ContactRequests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Request ID"
// @Success 200 {object} handlers.SuccessResponse
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 403 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/contact-requests/{id} [delete]
func (h *contactRequestHandler) DeleteContactRequest(c *fiber.Ctx) error {
	id := c.Params("id")
	claims, ok := middleware.ClaimsFromContext(c)
	if (len(claims.Roles) == 0) || !ok {
		return SendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	userId := claims.UserID
	userRole := claims.Roles[0]

	// Regular users can only delete their own contact requests
	if userRole == "user" {
		existingRequest, err := h.getContactRequestUseCase.Execute(c.Context(), id)
		if err != nil {
			return SendError(c, fiber.StatusInternalServerError, err.Error())
		}
		if existingRequest == nil {
			return SendError(c, fiber.StatusNotFound, "Contact request not found")
		}
		if existingRequest.SenderID != userId && existingRequest.ReceiverID != userId {
			return SendError(c, fiber.StatusForbidden, "Forbidden")
		}
	}

	err := h.deleteContactRequestUseCase.Execute(c.Context(), id)
	if err != nil {
		if err == contacts.ErrContactRequestNotFound {
			return SendError(c, fiber.StatusNotFound, err.Error())
		}
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}
	return SendSuccess(c, fiber.StatusOK, "Contact request deleted successfully", nil)
}

// extractTokenFromRequest pulls the JWT bearer token from Authorization header or cookies.
func extractTokenFromRequest(c *fiber.Ctx) string {
	// Try Authorization header first
	if auth := c.Get("Authorization"); auth != "" {
		const prefix = "Bearer "
		if len(auth) > len(prefix) && auth[:len(prefix)] == prefix {
			return auth[len(prefix):]
		}
	}
	// Fall back to access_token cookie
	if token := c.Cookies("access_token"); token != "" {
		return token
	}
	return ""
}
