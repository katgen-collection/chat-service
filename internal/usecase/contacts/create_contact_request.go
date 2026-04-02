package contacts

import (
	"context"
	"mikhailjbs/chat-service/internal/domain/contacts"
)

type CreateContactRequestUseCase interface {
	Execute(ctx context.Context, request *contacts.CreateContactRequest, bearerToken string) (*contacts.ContactRequest, error)
}

type createContactRequestUseCase struct {
	service contacts.Service
}

func NewCreateContactRequestUseCase(service contacts.Service) CreateContactRequestUseCase {
	return &createContactRequestUseCase{service: service}
}

func (uc *createContactRequestUseCase) Execute(ctx context.Context, request *contacts.CreateContactRequest, bearerToken string) (*contacts.ContactRequest, error) {
	return uc.service.CreateContactRequest(request, bearerToken)
}
