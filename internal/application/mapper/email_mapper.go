// Package mapper provides conversion functions between view documents and application models.
package mapper

import (
	"db_sync/internal/models"
	"db_sync/internal/view"
)

// mapImportantContactToEmail converts a single ImportantContactView to a models.Email.
func mapImportantContactToEmail(c view.ImportantContactView) models.Email {
	return models.Email{
		Address:    c.Email,
		Importance: c.Importance,
		Category:   c.Category,
	}
}

// MapImportantContacts converts a slice of ImportantContactView to a slice of models.Email.
func MapImportantContacts(contacts []view.ImportantContactView) []models.Email {
	res := make([]models.Email, len(contacts))
	for i, c := range contacts {
		res[i] = mapImportantContactToEmail(c)
	}
	return res
}

// mapImportantEmailToContact converts a single models.Email to an ImportantContactView.
func mapImportantEmailToContact(email models.Email) view.ImportantContactView {
	return view.ImportantContactView{
		Email:      email.Address,
		Category:   email.Category,
		Importance: email.Importance,
	}
}

// MapImportantEmails converts a slice of models.Email to a slice of ImportantContactView.
func MapImportantEmails(emails []models.Email) []view.ImportantContactView {
	res := make([]view.ImportantContactView, len(emails))
	for i, e := range emails {
		res[i] = mapImportantEmailToContact(e)
	}
	return res
}
