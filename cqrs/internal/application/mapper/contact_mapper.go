// Package mapper provides conversion functions between view documents and application models.
package mapper

import (
	"db_sync/internal/models"
	"db_sync/internal/view"
)

// mapImportantContactToContact converts a single ImportantContactView to a models.Contact.
func mapImportantContactToContact(c view.ImportantContactView) models.Contact {
	return models.Contact{
		Value:      c.Value,
		Importance: c.Importance,
		Category:   c.Category,
	}
}

// MapImportantContacts converts a slice of ImportantContactView to a slice of models.Contact.
func MapImportantContacts(contacts []view.ImportantContactView) []models.Contact {
	res := make([]models.Contact, len(contacts))
	for i, c := range contacts {
		res[i] = mapImportantContactToContact(c)
	}
	return res
}

// mapImportantContactModelToView converts a single models.Contact to an ImportantContactView.
func mapImportantContactModelToView(contact models.Contact) view.ImportantContactView {
	return view.ImportantContactView{
		Value:      contact.Value,
		Category:   contact.Category,
		Importance: contact.Importance,
	}
}

// MapImportantContactsToView converts a slice of models.Contact to a slice of ImportantContactView.
func MapImportantContactsToView(contacts []models.Contact) []view.ImportantContactView {
	res := make([]view.ImportantContactView, len(contacts))
	for i, c := range contacts {
		res[i] = mapImportantContactModelToView(c)
	}
	return res
}
