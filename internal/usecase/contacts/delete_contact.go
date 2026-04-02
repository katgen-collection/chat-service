package contacts

import (
	"context"
	"mikhailjbs/chat-service/internal/domain/contacts"
)

type DeleteContactUseCase interface {
	Execute(ctx context.Context, userID string, contactID string) error
}

type deleteContactUseCase struct {
	service contacts.Service
}

func NewDeleteContactUseCase(service contacts.Service) DeleteContactUseCase {
	return &deleteContactUseCase{service: service}
}

func (uc *deleteContactUseCase) Execute(ctx context.Context, userID string, contactID string) error {
	return uc.service.DeleteContact(userID, contactID)
}
