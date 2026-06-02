package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/multica-ai/multica/server/internal/handler"
)

func registerCRMRoutes(r chi.Router, h *handler.Handler) {
	// CRM
	r.Route("/api/crm", func(r chi.Router) {
		r.Get("/ai-settings", h.ListCRMAISettings)
		r.Get("/ai-history", h.ListCRMAIHistory)
		r.Put("/ai-settings/{automationKey}", h.UpdateCRMAISetting)
		r.Route("/accounts", func(r chi.Router) {
			r.Get("/", h.ListCRMAccounts)
			r.Post("/", h.CreateCRMAccount)
			r.Route("/{accountId}", func(r chi.Router) {
				r.Get("/", h.GetCRMAccount)
				r.Put("/", h.UpdateCRMAccount)
				r.Delete("/", h.DeleteCRMAccount)
				r.Get("/contacts", h.ListCRMContacts)
				r.Post("/contacts", h.CreateCRMContact)
				r.Route("/contacts/{contactId}", func(r chi.Router) {
					r.Put("/", h.UpdateCRMContact)
					r.Delete("/", h.DeleteCRMContact)
				})
				r.Get("/notes", h.ListCRMCommunicationNotes)
				r.Post("/notes", h.CreateCRMCommunicationNote)
				r.Post("/projects", h.LinkCRMAccountProject)
				r.Post("/follow-up-issues", h.CreateCRMFollowUpIssue)
				r.Get("/profile", h.GetCRMAccountProfile)
				r.Put("/profile", h.UpsertCRMAccountProfile)
				r.Post("/profile/suggestions", h.SuggestCRMAccountProfile)
				r.Post("/profile/suggestions/{suggestionId}/apply", h.ApplyCRMAccountProfileSuggestion)
			})
		})
		r.Route("/imap-settings", func(r chi.Router) {
			r.Get("/", h.ListCRMIMAPSettings)
			r.Put("/", h.UpsertCRMIMAPSetting)
			r.Post("/{mailboxId}/test", h.TestCRMIMAPSetting)
			r.Delete("/{mailboxId}", h.DeleteCRMIMAPSetting)
		})
		r.Get("/emailengine/status", h.GetCRMEmailEngineStatus)
		r.Get("/imap/sync-runs", h.ListCRMIMAPSyncRuns)
		r.Route("/reminders", func(r chi.Router) {
			r.Get("/", h.ListCRMReminders)
			r.Post("/", h.CreateCRMReminder)
			r.Patch("/{reminderId}", h.UpdateCRMReminderStatus)
		})
		r.Post("/imap/preview", h.PreviewCRMIMAP)
		r.Post("/imap/import", h.ImportCRMIMAP)
		r.Post("/imap/sync", h.SyncCRMIMAP)
		r.Route("/email-drafts", func(r chi.Router) {
			r.Get("/", h.ListCRMEmailDrafts)
			r.Post("/", h.CreateCRMEmailDraft)
			r.Post("/ai-suggest", h.SuggestCRMEmailDraftReply)
			r.Patch("/{draftId}", h.UpdateCRMEmailDraft)
			r.Post("/{draftId}/send", h.SendCRMEmailDraft)
		})
		r.Route("/email-threads", func(r chi.Router) {
			r.Get("/", h.ListCRMEmailThreads)
			r.Post("/", h.CreateCRMEmailThread)
			r.Route("/{threadId}", func(r chi.Router) {
				r.Get("/", h.GetCRMEmailThread)
				r.Get("/association-suggestions", h.SuggestCRMEmailThreadAssociations)
				r.Patch("/state", h.UpdateCRMEmailThreadState)
				r.Patch("/association", h.UpdateCRMEmailThreadAssociation)
				r.Get("/messages", h.ListCRMEmailMessages)
				r.Post("/messages", h.CreateCRMEmailMessage)
				r.Post("/trash", h.TrashCRMEmailThread)
				r.Post("/restore", h.RestoreCRMEmailThread)
				r.Post("/move-folder", h.MoveCRMEmailThread)
				r.Delete("/delete", h.DeleteCRMEmailThread)
			})
		})
		r.Get("/email-messages/{messageId}/attachment/{attachmentIndex}", h.ServeCRMEmailAttachment)

		r.Get("/imap/diagnostics", h.GetCRMIMAPDiagnostics)
		r.Post("/imap/test-connection", h.TestCRMIMAPConnection)
		r.Post("/imap/{mailboxId}/sync-cron", h.SetCRMIMAPSyncCron)
		r.Get("/sync-runs/errors", h.ListCRMIMAPSyncErrors)
	})
}
