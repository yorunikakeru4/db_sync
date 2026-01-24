package service

type SyncService struct {
	UserService    *UserService
	EmailService   *EmailService
	MessageService *MessageService
}

func NewSyncService(
	userService *UserService,
	emailService *EmailService,
	messageService *MessageService,
) *SyncService {
	return &SyncService{
		UserService:    userService,
		EmailService:   emailService,
		MessageService: messageService,
	}
}
