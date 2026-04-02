package handlers

import (
	"mikhailjbs/chat-service/internal/domain/contacts"
	usecase "mikhailjbs/chat-service/internal/usecase/contacts"
	"mikhailjbs/chat-service/internal/infra/middleware"

	"github.com/gofiber/fiber/v2"
)

type ContactHandler interface {
	GetContact(c *fiber.Ctx) error
	GetContacts(c *fiber.Ctx) error
	UpdateContact(c *fiber.Ctx) error
	DeleteContact(c *fiber.Ctx) error
}

type contactHandler struct {
	getContactUseCase    usecase.GetContactUseCase
	getContactsUseCase   usecase.GetContactsUseCase
	updateContactUseCase usecase.UpdateContactUseCase
	deleteContactUseCase usecase.DeleteContactUseCase
}

func NewContactHandler(
	getContactUseCase    usecase.GetContactUseCase,
	getContactsUseCase   usecase.GetContactsUseCase,
	updateContactUseCase usecase.UpdateContactUseCase,
	deleteContactUseCase usecase.DeleteContactUseCase,
) ContactHandler {
	return &contactHandler{
		getContactUseCase:    getContactUseCase,
		getContactsUseCase:   getContactsUseCase,
		updateContactUseCase: updateContactUseCase,
		deleteContactUseCase: deleteContactUseCase,
	}
}

// GetContact godoc
// @Summary Get contact details
// @Description Fetches details of a specific contact by ID.
// @Tags Contacts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Contact ID"
// @Success 200 {object} handlers.SuccessResponse{data=contacts.Contact}
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 403 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/contacts/{id} [get]
func (h *contactHandler) GetContact(c *fiber.Ctx) error {
	id := c.Params("id")
	claims, ok := middleware.ClaimsFromContext(c)
	if (len(claims.Roles) == 0) || !ok {
		return SendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	userId := claims.UserID
	userRole := claims.Roles[0]

	contact, err := h.getContactUseCase.Execute(c.Context(), id)
	if err != nil {
		if err == contacts.ErrContactNotFound {
			return SendError(c, fiber.StatusNotFound, err.Error())
		}
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Regular users can only access their own contacts
	if userRole == "user" {
		if contact.UserID != userId && contact.ContactID != userId {
			return SendError(c, fiber.StatusForbidden, "Forbidden")
		}
	}

	return SendSuccess(c, fiber.StatusOK, "Contact  retrieved successfully", contact)
}

// GetContacts godoc
// @Summary List contacts
// @Description Retrieve a list of contacts for the authenticated user with optional filtering.
// @Tags Contacts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search by name"
// @Param limit query int false "Pagination limit"
// @Param offset query int false "Pagination offset"
// @Param sort_by query string false "Field to sort by"
// @Param order query string false "Sort order (asc/desc)"
// @Success 200 {object} handlers.SuccessResponse{data=[]contacts.Contact}
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/contacts [get]
func (h *contactHandler) GetContacts(c *fiber.Ctx) error {
	claims, ok := middleware.ClaimsFromContext(c)
	if (len(claims.Roles) == 0) || !ok {
		return SendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	userId := claims.UserID
	userRole := claims.Roles[0]

	var queryParams contacts.ContactQueryParams
	if err := c.QueryParser(&queryParams); err != nil {
		return SendError(c, fiber.StatusBadRequest, "Invalid query parameters")
	}

	// Regular users can only see their own contacts
	if userRole == "user" {
		queryParams.UserID = userId
	}

	contacts, err := h.getContactsUseCase.Execute(c.Context(), &queryParams)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, "Contact s retrieved successfully", contacts)
}	

// UpdateContact godoc
// @Summary Update contact
// @Description Updates contact preferences like muted status or assigned name.
// @Tags Contacts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Contact ID"
// @Param request body contacts.UpdateContact true "Update Contact Request"
// @Success 200 {object} handlers.SuccessResponse{data=contacts.Contact}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 403 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/contacts/{id} [put]
func (h *contactHandler) UpdateContact(c *fiber.Ctx) error {
	id := c.Params("id")
	claims, ok := middleware.ClaimsFromContext(c)
	if (len(claims.Roles) == 0) || !ok {
		return SendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	userId := claims.UserID
	userRole := claims.Roles[0]

	var req contacts.UpdateContact
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "Invalid  body")
	}

	// Regular users can only update their own contacts
	if userRole == "user" {
		existingContact, err := h.getContactUseCase.Execute(c.Context(), id)
		if err != nil {
			return SendError(c, fiber.StatusInternalServerError, err.Error())
		}
		if existingContact == nil {
			return SendError(c, fiber.StatusNotFound, "Contact not found")
		}
		if existingContact.UserID != userId {
			return SendError(c, fiber.StatusForbidden, "Forbidden")
		}
	}

	updatedContact, err := h.updateContactUseCase.Execute(c.Context(), &req, id)
	if err != nil {
		if err == contacts.ErrContactNotFound {
			return SendError(c, fiber.StatusNotFound, err.Error())
		}
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}
	return SendSuccess(c, fiber.StatusOK, "Contact  updated successfully", updatedContact)
}

// DeleteContact godoc
// @Summary Delete contact
// @Description Removes a contact relationship.
// @Tags Contacts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Contact ID"
// @Success 200 {object} handlers.SuccessResponse
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/contacts/{id} [delete]
func (h *contactHandler) DeleteContact(c *fiber.Ctx) error {
	id := c.Params("id")
	claims, ok := middleware.ClaimsFromContext(c)
	if (len(claims.Roles) == 0) || !ok {
		return SendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	userId := claims.UserID

	err := h.deleteContactUseCase.Execute(c.Context(), userId, id) // fix after middleware
	if err != nil {
		if err == contacts.ErrContactNotFound {
			return SendError(c, fiber.StatusNotFound, err.Error())
		}
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}
	return SendSuccess(c, fiber.StatusOK, "Contact  deleted successfully", nil)
}