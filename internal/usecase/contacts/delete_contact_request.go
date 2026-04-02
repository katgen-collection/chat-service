package contacts

import (
	"context"
	"mikhailjbs/chat-service/internal/domain/contacts"
)

type DeleteContactRequestUseCase interface {
	Execute(ctx context.Context, id string) error
}

type deleteContactRequestUseCase struct {
	service contacts.Service
}

func NewDeleteContactRequestUseCase(service contacts.Service) DeleteContactRequestUseCase {
	return &deleteContactRequestUseCase{service: service}
}

func (uc *deleteContactRequestUseCase) Execute(ctx context.Context, id string) error {
	return uc.service.DeleteContactRequest(id)
}
