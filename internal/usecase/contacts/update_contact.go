package contacts

import (
	"context"
	"mikhailjbs/chat-service/internal/domain/contacts"
)

type UpdateContactUseCase interface {
	Execute(ctx context.Context, contact *contacts.UpdateContact, id string) (*contacts.Contact, error)
}

type updateContactUseCase struct {
	service contacts.Service
}

func NewUpdateContactUseCase(service contacts.Service) UpdateContactUseCase {
	return &updateContactUseCase{service: service}
}

func (uc *updateContactUseCase) Execute(ctx context.Context, contact *contacts.UpdateContact, id string) (*contacts.Contact, error) {
	return uc.service.UpdateContact(contact, id)
}
