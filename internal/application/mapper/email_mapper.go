package mapper

import (
	"db_sync/internal/models"
	"db_sync/internal/view"
)

func mapImportantContactToEmail(c view.ImportantContactView) models.Email {
	return models.Email{
		Address:    c.Email,
		Importance: c.Importance,
		Category:   c.Category,
	}
}

func MapImportantContacts(contacts []view.ImportantContactView) []models.Email {
	res := make([]models.Email, len(contacts))
	for i, c := range contacts {
		res[i] = mapImportantContactToEmail(c)
	}
	return res
}

func mapImportantEmailToContact(email models.Email) view.ImportantContactView {
	return view.ImportantContactView{
		Email:      email.Address,
		Category:   email.Category,
		Importance: email.Importance,
	}
}

func MapImportantEmails(emails []models.Email) []view.ImportantContactView {
	res := make([]view.ImportantContactView, len(emails))
	for i, e := range emails {
		res[i] = mapImportantEmailToContact(e)
	}
	return res
}
