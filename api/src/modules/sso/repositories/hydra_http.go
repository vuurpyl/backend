package repositories

import (
	"context"
	"net/url"

	"github.com/volatiletech/null"
	"gitlab.misakey.dev/misakey/backend/api/src/modules/sso/domain/consent"
	"gitlab.misakey.dev/misakey/backend/api/src/modules/sso/domain/login"
	"gitlab.misakey.dev/misakey/msk-sdk-go/merror"
	"gitlab.misakey.dev/misakey/msk-sdk-go/rester"
)

// HTTP implements Hydra repository interface using HTTP REST
type HydraHTTP struct {
	publicRester         rester.Client
	authPublicFormRester rester.Client
	adminRester          rester.Client
	formAdminHydraRester rester.Client
}

// NewHydraHTTP is HTTP hydra structure constructor
func NewHydraHTTP(
	publicRester rester.Client,
	authPublicFormRester rester.Client,
	adminRester rester.Client,
	formAdminHydraRester rester.Client,
) *HydraHTTP {
	return &HydraHTTP{
		publicRester:         publicRester,
		authPublicFormRester: authPublicFormRester,
		adminRester:          adminRester,
		formAdminHydraRester: formAdminHydraRester,
	}
}

// GetLoginContext from hydra
func (hh HydraHTTP) GetLoginContext(ctx context.Context, loginChallenge string) (login.Context, error) {
	// 1. prepare the request
	// expected hydra DTO format
	hydraLogReq := struct {
		Challenge      string   `json:"challenge"`
		Skip           bool     `json:"skip"`
		Subject        string   `json:"subject"`
		RequestedScope []string `json:"requested_scope"`
		Client         struct { // concerned relying party
			ID      string `json:"client_id"`
			Name    string `json:"client_name"`
			LogoURI string `json:"logo_uri"`
		} `json:"client"`
		OIDCContext struct { // OIDC context of the current request
			ACRValues []string `json:"acr_values"`
			LoginHint string   `json:"login_hint"`
		} `json:"oidc_context"`
	}{}
	// query parameters
	params := url.Values{}
	params.Add("login_challenge", loginChallenge)

	// 2. perform the request
	logCtx := login.Context{}
	err := hh.adminRester.Get(ctx, "/oauth2/auth/requests/login", params, &hydraLogReq)
	if err != nil {
		if merror.HasCode(err, merror.NotFoundCode) {
			err = merror.Transform(err).Detail("challenge", merror.DVNotFound)
		}
		return logCtx, err
	}

	// 3. fill domain model using the DTO
	logCtx.Challenge = hydraLogReq.Challenge
	logCtx.Skip = hydraLogReq.Skip
	logCtx.Subject = hydraLogReq.Subject
	logCtx.RequestedScope = hydraLogReq.RequestedScope
	logCtx.Client.ID = hydraLogReq.Client.ID
	logCtx.Client.Name = hydraLogReq.Client.Name
	if hydraLogReq.Client.LogoURI != "" {
		logCtx.Client.LogoURL = null.StringFrom(hydraLogReq.Client.LogoURI)
	}
	logCtx.OIDCContext.ACRValues = hydraLogReq.OIDCContext.ACRValues
	logCtx.OIDCContext.LoginHint = hydraLogReq.OIDCContext.LoginHint
	return logCtx, nil
}

// Login user to hydra
func (hh HydraHTTP) Login(ctx context.Context, loginChallenge string, acceptance login.Acceptance) (login.Redirect, error) {
	redirect := login.Redirect{}
	params := url.Values{}
	params.Add("login_challenge", loginChallenge)
	err := hh.adminRester.Put(ctx, "/oauth2/auth/requests/login/accept", params, acceptance, &redirect)
	if err != nil {
		if merror.HasCode(err, merror.NotFoundCode) {
			err = merror.Transform(err).Detail("challenge", merror.DVNotFound)
		}
		return redirect, err
	}
	return redirect, nil
}

// GetConsentContext from hydra
func (h *HydraHTTP) GetConsentContext(ctx context.Context, consentChallenge string) (consent.Context, error) {
	consentCtx := consent.Context{}
	params := url.Values{}
	params.Add("consent_challenge", consentChallenge)
	err := h.adminRester.Get(ctx, "/oauth2/auth/requests/consent", params, &consentCtx)
	if err != nil {
		return consentCtx, err
	}
	return consentCtx, nil
}

// Consent user's scope to hydra
func (h *HydraHTTP) Consent(ctx context.Context, consentChallenge string, acceptance consent.Acceptance) (consent.Redirect, error) {
	redirect := consent.Redirect{}
	params := url.Values{}
	params.Add("consent_challenge", consentChallenge)
	err := h.adminRester.Put(ctx, "/oauth2/auth/requests/consent/accept", params, acceptance, &redirect)
	if err != nil {
		return redirect, err
	}
	return redirect, nil
}

// // Logout : invalidates a subject's authentication session
// func (h *HydraHTTP) Logout(ctx context.Context, id string) error {
// 	route := fmt.Sprintf("/oauth2/auth/sessions/login?subject=%s", id)
//
// 	return afs.adminRester.Delete(ctx, route, nil)
// }

// // RevokeToken : invalidate access & refresh tokens
// func (h *HydraHTTP) RevokeToken(ctx context.Context, revocation model.TokenRevocation) error {
// 	params := url.Values{}
// 	params.Add("token", revocation.Token)
// 	params.Add("client_id", revocation.ClientID)
// 	params.Add("client_secret", revocation.ClientSecret)
// 	return afs.authPublicFormRester.Post(ctx, "/oauth2/revoke", nil, params, nil)
// }
//
// func (h *HydraHTTP) Introspect(ctx context.Context, opaqueTok string) (*model.IntrospectedToken, error) {
// 	introTok := model.IntrospectedToken{}
// 	route := fmt.Sprintf("/oauth2/introspect")
//
// 	params := url.Values{}
// 	params.Add("token", opaqueTok)
//
// 	err := afs.formAdminHydraRester.Post(ctx, route, nil, params, &introTok)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &introTok, nil
// }
//
// // CreateClient: create a new Hydra Client in hydra
// func (h *HydraHTTP) CreateClient(ctx context.Context, hydraClient *model.HydraClient) error {
// 	route := fmt.Sprintf("/clients")
//
// 	return afs.adminRester.Post(ctx, route, nil, hydraClient, nil)
// }
//

//
// // UpdateClient: update Hydra Client in hydra
// func (h *HydraHTTP) UpdateClient(ctx context.Context, hydraClient *model.HydraClient) error {
// 	route := fmt.Sprintf("/clients/%s", hydraClient.ID)
//
// 	return afs.adminRester.Put(ctx, route, nil, hydraClient, hydraClient)
// }