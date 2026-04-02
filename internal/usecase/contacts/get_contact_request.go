package contacts

import (
	"context"
	"mikhailjbs/chat-service/internal/domain/contacts"
)

type GetContactRequestUseCase interface {
	Execute(ctx context.Context, id string) (*contacts.ContactRequest, error)
}

type getContactRequestUseCase struct {
	service contacts.Service
}

func NewGetContactRequestUseCase(service contacts.Service) GetContactRequestUseCase {
	return &getContactRequestUseCase{service: service}
}

func (uc *getContactRequestUseCase) Execute(ctx context.Context, id string) (*contacts.ContactRequest, error) {
	return uc.service.GetContactRequestByID(id)
}
