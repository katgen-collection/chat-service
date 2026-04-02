package contacts

import (
	"context"
	"mikhailjbs/chat-service/internal/domain/contacts"
)

type UpdateContactRequestUseCase interface {
	Execute(ctx context.Context, id string, update *contacts.UpdateContactRequest, bearerToken string) (*contacts.ContactRequest, error)
}

type updateContactRequestUseCase struct {
	service contacts.Service
}

func NewUpdateContactRequestUseCase(service contacts.Service) UpdateContactRequestUseCase {
	return &updateContactRequestUseCase{service: service}
}

func (uc *updateContactRequestUseCase) Execute(ctx context.Context, id string, update *contacts.UpdateContactRequest, bearerToken string) (*contacts.ContactRequest, error) {
	return uc.service.UpdateContactRequest(update, id, bearerToken)
}
