package api

import (
	"fmt"

	"github.com/go-chi/chi"
	"gitlab.com/distributed_lab/ape"

	"github.com/rarimo/humanornot-svc/internal/service/api/handlers"
	"github.com/rarimo/humanornot-svc/internal/service/api/requests"
)

func (s *service) router() chi.Router {
	r := chi.NewRouter()

	r.Use(
		ape.RecoverMiddleware(s.log),
		ape.LoganMiddleware(s.log),
		ape.CtxMiddleware(
			handlers.CtxLog(s.log),
			handlers.CtxHumanornotSvc(s.humanornotSvc),
		),
	)

	r.Route("/integrations/humanornot-svc", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Route("/public", func(r chi.Router) {
				r.Post(fmt.Sprintf("/verify/{%s}", requests.IdentityProviderPathParam), handlers.Verify)
				r.Post("/nonce", handlers.GetNonce)
				r.Get(fmt.Sprintf("/{%s}/provider", requests.GetProviderByIdentityIdPathParam), handlers.GetProviderByIdentityId)
				r.Get(fmt.Sprintf("/status/{%s}", requests.VerifyIDPathParam), handlers.GetVerifyStatus)
			})
		})
	})

	return r
}
