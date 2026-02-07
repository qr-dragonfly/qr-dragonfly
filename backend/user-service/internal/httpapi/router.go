package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/smithy-go"
	"github.com/stripe/stripe-go/v81"

	"user-service/internal/cognito"
	"user-service/internal/middleware"
	"user-service/internal/model"
)

func smithyErrorCode(err error) string {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return strings.TrimSpace(apiErr.ErrorCode())
	}
	return ""
}

func writeAuthError(w http.ResponseWriter, r *http.Request, status int, fallback string, err error) {
	code := smithyErrorCode(err)
	log.Printf("auth error request_id=%s code=%s err=%v", r.Header.Get("X-Request-Id"), code, err)
	if code == "" {
		writeJSON(w, status, map[string]string{"error": fallback})
		return
	}
	// Prefer a stable, client-friendly error string.
	switch code {
	case "UserNotConfirmedException":
		writeJSON(w, status, map[string]string{"error": "user_not_confirmed"})
	case "UsernameExistsException":
		writeJSON(w, status, map[string]string{"error": "user_already_exists"})
	case "CodeMismatchException":
		writeJSON(w, status, map[string]string{"error": "code_mismatch"})
	case "ExpiredCodeException":
		writeJSON(w, status, map[string]string{"error": "code_expired"})
	case "LimitExceededException":
		writeJSON(w, status, map[string]string{"error": "rate_limited"})
	case "NotAuthorizedException":
		writeJSON(w, status, map[string]string{"error": "not_authorized"})
	case "InvalidPasswordException":
		writeJSON(w, status, map[string]string{"error": "invalid_password"})
	case "InvalidParameterException":
		// Provide a more actionable error for common Cognito misconfiguration.
		if strings.Contains(err.Error(), "USER_PASSWORD_AUTH flow not enabled") {
			writeJSON(w, status, map[string]string{"error": "password_auth_not_enabled"})
			return
		}
		writeJSON(w, status, map[string]string{"error": "invalid_parameter"})
	default:
		// Preserve the upstream code (lowercase) for debugging, but keep it stable.
		writeJSON(w, status, map[string]string{"error": strings.ToLower(code)})
	}
}

type Server struct {
	Cognito cognito.API

	UserPoolID string
	ClientID   string
	// Optional; required if the App Client has a client secret.
	ClientSecret string

	AdminAPIKey string

	CookieSecure   bool
	CookieSameSite http.SameSite

	// Stripe integration (optional)
	StripeClient interface {
		CreateCheckoutSession(customerEmail string, priceID string) (*stripe.CheckoutSession, error)
		CreateSubscriptionWithPaymentMethod(customerEmail, paymentMethodID, priceID string) (*stripe.Subscription, error)
		CreateCustomerPortalSession(customerEmail string) (*stripe.BillingPortalSession, error)
		ConstructEvent(payload []byte, signature string) (stripe.Event, error)
		GetPriceIDForPlan(plan string) (string, error)
	}
}

const cognitoUserTypeAttr = "custom:user_type"
const cognitoEntitlementsAttr = "custom:entitlements"

func normalizeUserType(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

// mapUserTypeToEntitlement maps legacy user_type to entitlements format
func mapUserTypeToEntitlement(userType string) string {
	userType = normalizeUserType(userType)
	switch userType {
	case "free":
		return "free"
	case "basic":
		return "basic"
	case "enterprise":
		return "enterprise"
	case "admin":
		return "admin"
	default:
		return "free"
	}
}

// mapSubscriptionToEntitlement determines entitlement from Stripe subscription
func mapSubscriptionToEntitlement(priceID, basicPriceID, enterprisePriceID string) string {
	switch priceID {
	case basicPriceID:
		return "basic"
	case enterprisePriceID:
		return "enterprise"
	default:
		return "free"
	}
}

func derivedUsernameFromEmail(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))
	sum := sha256.Sum256([]byte(email))
	// Cognito allows a broad set of chars in Username, but a hex string is safe.
	// Prefix to avoid any future ambiguity.
	return "email_" + hex.EncodeToString(sum[:])
}

func derivedUsernameFromIdentifier(id string) string {
	if !strings.Contains(id, "@") {
		return ""
	}
	derived := derivedUsernameFromEmail(id)
	if derived == "" || derived == id {
		return ""
	}
	return derived
}

func shouldTryDerivedUsername(err error) bool {
	code := smithyErrorCode(err)
	return usernameCannotBeEmailInThisPool(err) || code == "UserNotFoundException"
}

func usernameCannotBeEmailInThisPool(err error) bool {
	// Observed error when the User Pool is configured with email alias:
	// "Username cannot be of email format, since user pool is configured for email alias."
	return smithyErrorCode(err) == "InvalidParameterException" && strings.Contains(err.Error(), "Username cannot be of email format")
}

func userTypeAttributeNotInSchema(err error) bool {
	// Observed when the User Pool schema doesn't include custom:user_type:
	// "Attributes did not conform to the schema: Type for attribute {custom:user_type} could not be determined"
	return smithyErrorCode(err) == "InvalidParameterException" && strings.Contains(err.Error(), "custom:user_type") && strings.Contains(err.Error(), "did not conform to the schema")
}

func isAllowedUserType(value string, allowAdmin bool) bool {
	switch value {
	case "free", "basic", "enterprise":
		return true
	case "admin":
		return allowAdmin
	default:
		return false
	}
}

func NewRouter(srv Server) http.Handler {
	mux := http.NewServeMux()

	wrap := func(h http.Handler) http.Handler {
		return middleware.Recoverer(middleware.RequestID(middleware.ExposeResponseHeaders(middleware.EnforceJSONHandler(h))))
	}

	healthHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	registerHandler := http.HandlerFunc(srv.handleRegister)
	loginHandler := http.HandlerFunc(srv.handleLogin)
	logoutHandler := http.HandlerFunc(srv.handleLogout)
	meHandler := http.HandlerFunc(srv.handleMe)
	confirmHandler := http.HandlerFunc(srv.handleConfirmSignUp)
	resendConfirmationHandler := http.HandlerFunc(srv.handleResendConfirmation)
	forgotPasswordHandler := http.HandlerFunc(srv.handleForgotPassword)
	confirmForgotPasswordHandler := http.HandlerFunc(srv.handleConfirmForgotPassword)
	changePasswordHandler := http.HandlerFunc(srv.handleChangePassword)

	adminCollectionHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			srv.handleAdminListUsers(w, r)
			return
		case http.MethodPost:
			srv.handleAdminCreateUser(w, r)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})

	adminItemHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This handler is registered for "/api/users/".
		// More specific patterns (register/login/logout/me) will win in ServeMux.
		rest := strings.TrimPrefix(r.URL.Path, "/api/users/")
		rest = strings.Trim(rest, "/")
		if rest == "" {
			// Allow trailing-slash collection requests.
			adminCollectionHandler.ServeHTTP(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			srv.handleAdminGetUser(w, r)
			return
		case http.MethodPatch:
			srv.handleAdminUpdateUser(w, r)
			return
		case http.MethodDelete:
			srv.handleAdminDeleteUser(w, r)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})

	mux.Handle("/healthz", wrap(healthHandler))

	mux.Handle("/api/users/register", wrap(registerHandler))
	mux.Handle("/api/users/login", wrap(loginHandler))
	mux.Handle("/api/users/logout", wrap(logoutHandler))
	mux.Handle("/api/users/me", wrap(meHandler))
	mux.Handle("/api/users/confirm", wrap(confirmHandler))
	mux.Handle("/api/users/resend-confirmation", wrap(resendConfirmationHandler))
	mux.Handle("/api/users/forgot-password", wrap(forgotPasswordHandler))
	mux.Handle("/api/users/confirm-forgot-password", wrap(confirmForgotPasswordHandler))
	mux.Handle("/api/users/change-password", wrap(changePasswordHandler))

	// Admin-style CRUD (guarded)
	mux.Handle("/api/users", wrap(http.HandlerFunc(requireAdmin(srv.AdminAPIKey, adminCollectionHandler))))
	mux.Handle("/api/users/", wrap(http.HandlerFunc(requireAdmin(srv.AdminAPIKey, adminItemHandler))))

	// Stripe routes (if Stripe is configured)
	if srv.StripeClient != nil {
		checkoutHandler := http.HandlerFunc(srv.handleCreateCheckoutSession)
		subscriptionHandler := http.HandlerFunc(srv.handleCreateSubscription)
		portalHandler := http.HandlerFunc(srv.handleCreatePortalSession)
		webhookHandler := http.HandlerFunc(srv.handleStripeWebhook)
		mux.Handle("/api/stripe/checkout-session", wrap(checkoutHandler))
		mux.Handle("/api/stripe/subscription", wrap(subscriptionHandler))
		mux.Handle("/api/stripe/portal-session", wrap(portalHandler))
		mux.Handle("/api/stripe/webhook", wrap(webhookHandler))
	}

	return mux
}

func (srv Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req createUserInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Password = strings.TrimSpace(req.Password)
	req.UserType = normalizeUserType(req.UserType)
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email_and_password_required"})
		return
	}
	if req.UserType == "" {
		req.UserType = "free"
	}
	// Prevent self-registration from setting admin.
	if !isAllowedUserType(req.UserType, false) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_user_type"})
		return
	}

	attrsBase := []types.AttributeType{{Name: aws.String("email"), Value: aws.String(req.Email)}}

	attrsWithUserType := attrsBase
	if req.UserType != "" {
		attrsWithUserType = append(attrsWithUserType, types.AttributeType{Name: aws.String(cognitoUserTypeAttr), Value: aws.String(req.UserType)})
	}

	signUp := func(username string, attrs []types.AttributeType) (*cognitoidentityprovider.SignUpOutput, error) {
		in := &cognitoidentityprovider.SignUpInput{
			ClientId:       aws.String(srv.ClientID),
			Username:       aws.String(username),
			Password:       aws.String(req.Password),
			UserAttributes: attrs,
		}
		if srv.ClientSecret != "" {
			in.SecretHash = aws.String(cognito.SecretHash(username, srv.ClientID, srv.ClientSecret))
		}
		return srv.Cognito.SignUp(ctx, in)
	}

	// Use the derived username scheme so we're consistent with pools configured
	// to allow email as an alias while disallowing email-format usernames.
	username := derivedUsernameFromEmail(req.Email)
	attrs := attrsWithUserType
	out, err := signUp(username, attrs)
	if err != nil && userTypeAttributeNotInSchema(err) && len(attrs) != len(attrsBase) {
		attrs = attrsBase
		out, err = signUp(username, attrs)
	}
	if err != nil {
		writeAuthError(w, r, http.StatusBadRequest, "signup_failed", err)
		return
	}

	session := AuthSession{User: model.User{ID: aws.ToString(out.UserSub), Email: req.Email, UserType: req.UserType}.NormalizeForResponse()}
	writeJSON(w, http.StatusOK, session)
}

func (srv Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req loginInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Password = strings.TrimSpace(req.Password)
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email_and_password_required"})
		return
	}

	attempt := func(username string) (*cognitoidentityprovider.InitiateAuthOutput, error) {
		params := map[string]string{
			"USERNAME": username,
			"PASSWORD": req.Password,
		}
		if srv.ClientSecret != "" {
			params["SECRET_HASH"] = cognito.SecretHash(username, srv.ClientID, srv.ClientSecret)
		}

		return srv.Cognito.InitiateAuth(ctx, &cognitoidentityprovider.InitiateAuthInput{
			AuthFlow:       types.AuthFlowTypeUserPasswordAuth,
			ClientId:       aws.String(srv.ClientID),
			AuthParameters: params,
		})
	}

	derived := derivedUsernameFromEmail(req.Email)
	authOut, err := attempt(derived)
	if err != nil {
		// Back-compat: if some users were created with email as the Username,
		// try the raw email only when the derived username isn't found.
		if smithyErrorCode(err) == "UserNotFoundException" {
			if authOut2, err2 := attempt(req.Email); err2 == nil {
				authOut = authOut2
				err = nil
			} else {
				err = err2
			}
		}
	}
	if err != nil {
		writeAuthError(w, r, http.StatusUnauthorized, "login_failed", err)
		return
	}
	if authOut.AuthenticationResult == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login_failed"})
		return
	}

	access := aws.ToString(authOut.AuthenticationResult.AccessToken)
	idToken := aws.ToString(authOut.AuthenticationResult.IdToken)
	refresh := aws.ToString(authOut.AuthenticationResult.RefreshToken)

	if access != "" {
		setCookie(w, "access_token", access, srv.CookieSecure, srv.CookieSameSite)
	}
	if idToken != "" {
		setCookie(w, "id_token", idToken, srv.CookieSecure, srv.CookieSameSite)
	}
	if refresh != "" {
		setCookie(w, "refresh_token", refresh, srv.CookieSecure, srv.CookieSameSite)
	}

	user, err := getUserFromAccessToken(ctx, srv.Cognito, access)
	if err != nil {
		// still return token, but without user details
		writeJSON(w, http.StatusOK, AuthSession{User: model.User{ID: req.Email, Email: req.Email}.NormalizeForResponse(), Token: idToken})
		return
	}

	writeJSON(w, http.StatusOK, AuthSession{User: user.NormalizeForResponse(), Token: idToken})
}

func (srv Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	access, _ := readCookie(r, "access_token")
	if access != "" {
		_, _ = srv.Cognito.GlobalSignOut(ctx, &cognitoidentityprovider.GlobalSignOutInput{AccessToken: aws.String(access)})
	}
	clearCookie(w, "access_token", srv.CookieSecure, srv.CookieSameSite)
	clearCookie(w, "id_token", srv.CookieSecure, srv.CookieSameSite)
	clearCookie(w, "refresh_token", srv.CookieSecure, srv.CookieSameSite)
	w.WriteHeader(http.StatusNoContent)
}

func (srv Server) handleMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	access, _ := readCookie(r, "access_token")
	if access == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not_authenticated"})
		return
	}

	user, err := getUserFromAccessToken(ctx, srv.Cognito, access)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not_authenticated"})
		return
	}
	writeJSON(w, http.StatusOK, user.NormalizeForResponse())
}

func (srv Server) handleConfirmSignUp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req confirmSignUpInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Code = strings.TrimSpace(req.Code)
	if req.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email_required"})
		return
	}
	if req.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code_required"})
		return
	}

	username := derivedUsernameFromEmail(req.Email)
	in := &cognitoidentityprovider.ConfirmSignUpInput{
		ClientId:           aws.String(srv.ClientID),
		Username:           aws.String(username),
		ConfirmationCode:   aws.String(req.Code),
		ForceAliasCreation: false,
	}
	if srv.ClientSecret != "" {
		in.SecretHash = aws.String(cognito.SecretHash(username, srv.ClientID, srv.ClientSecret))
	}

	if _, err := srv.Cognito.ConfirmSignUp(ctx, in); err != nil {
		// Back-compat: try email username only if derived isn't found.
		if smithyErrorCode(err) == "UserNotFoundException" {
			username = req.Email
			in.Username = aws.String(username)
			if srv.ClientSecret != "" {
				in.SecretHash = aws.String(cognito.SecretHash(username, srv.ClientID, srv.ClientSecret))
			}
			if _, err2 := srv.Cognito.ConfirmSignUp(ctx, in); err2 == nil {
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			} else {
				err = err2
			}
		}
		writeAuthError(w, r, http.StatusBadRequest, "confirm_failed", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (srv Server) handleResendConfirmation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req resendConfirmationInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email_required"})
		return
	}

	derived := derivedUsernameFromEmail(req.Email)
	username := derived
	in := &cognitoidentityprovider.ResendConfirmationCodeInput{
		ClientId: aws.String(srv.ClientID),
		Username: aws.String(username),
	}
	if srv.ClientSecret != "" {
		in.SecretHash = aws.String(cognito.SecretHash(username, srv.ClientID, srv.ClientSecret))
	}

	out, err := srv.Cognito.ResendConfirmationCode(ctx, in)
	if err != nil {
		// Back-compat: try email username only if derived isn't found.
		if smithyErrorCode(err) == "UserNotFoundException" {
			username = req.Email
			in.Username = aws.String(username)
			if srv.ClientSecret != "" {
				in.SecretHash = aws.String(cognito.SecretHash(username, srv.ClientID, srv.ClientSecret))
			}
			if out2, err2 := srv.Cognito.ResendConfirmationCode(ctx, in); err2 == nil {
				out = out2
				err = nil
			} else {
				err = err2
			}
		}
		if err != nil {
			writeAuthError(w, r, http.StatusBadRequest, "resend_failed", err)
			return
		}
	}

	resp := map[string]any{"status": "ok"}
	if out != nil && out.CodeDeliveryDetails != nil {
		d := out.CodeDeliveryDetails
		delivery := map[string]string{}
		if v := aws.ToString(d.Destination); v != "" {
			delivery["destination"] = v
		}
		if v := string(d.DeliveryMedium); v != "" {
			delivery["medium"] = v
		}
		if v := aws.ToString(d.AttributeName); v != "" {
			delivery["attribute"] = v
		}
		if len(delivery) > 0 {
			resp["delivery"] = delivery
			log.Printf("resend_confirmation request_id=%s delivery=%v", r.Header.Get("X-Request-Id"), delivery)
		} else {
			log.Printf("resend_confirmation request_id=%s delivery=empty", r.Header.Get("X-Request-Id"))
		}
	} else {
		log.Printf("resend_confirmation request_id=%s delivery=none", r.Header.Get("X-Request-Id"))
	}

	writeJSON(w, http.StatusOK, resp)
}

func (srv Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req forgotPasswordInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email_required"})
		return
	}

	username := derivedUsernameFromEmail(req.Email)
	in := &cognitoidentityprovider.ForgotPasswordInput{
		ClientId: aws.String(srv.ClientID),
		Username: aws.String(username),
	}
	if srv.ClientSecret != "" {
		in.SecretHash = aws.String(cognito.SecretHash(username, srv.ClientID, srv.ClientSecret))
	}

	if _, err := srv.Cognito.ForgotPassword(ctx, in); err != nil {
		// Back-compat: try email username only if derived isn't found.
		if smithyErrorCode(err) == "UserNotFoundException" {
			username = req.Email
			in.Username = aws.String(username)
			if srv.ClientSecret != "" {
				in.SecretHash = aws.String(cognito.SecretHash(username, srv.ClientID, srv.ClientSecret))
			}
			if _, err2 := srv.Cognito.ForgotPassword(ctx, in); err2 == nil {
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			} else {
				err = err2
			}
		}
		writeAuthError(w, r, http.StatusBadRequest, "forgot_failed", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (srv Server) handleConfirmForgotPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req confirmForgotPasswordInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Code = strings.TrimSpace(req.Code)
	req.NewPassword = strings.TrimSpace(req.NewPassword)
	if req.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email_required"})
		return
	}
	if req.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code_required"})
		return
	}
	if req.NewPassword == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password_required"})
		return
	}

	username := derivedUsernameFromEmail(req.Email)
	in := &cognitoidentityprovider.ConfirmForgotPasswordInput{
		ClientId:         aws.String(srv.ClientID),
		Username:         aws.String(username),
		ConfirmationCode: aws.String(req.Code),
		Password:         aws.String(req.NewPassword),
	}
	if srv.ClientSecret != "" {
		in.SecretHash = aws.String(cognito.SecretHash(username, srv.ClientID, srv.ClientSecret))
	}

	if _, err := srv.Cognito.ConfirmForgotPassword(ctx, in); err != nil {
		// Back-compat: try email username only if derived isn't found.
		if smithyErrorCode(err) == "UserNotFoundException" {
			username = req.Email
			in.Username = aws.String(username)
			if srv.ClientSecret != "" {
				in.SecretHash = aws.String(cognito.SecretHash(username, srv.ClientID, srv.ClientSecret))
			}
			if _, err2 := srv.Cognito.ConfirmForgotPassword(ctx, in); err2 == nil {
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			} else {
				err = err2
			}
		}
		writeAuthError(w, r, http.StatusBadRequest, "reset_failed", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (srv Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	access, _ := readCookie(r, "access_token")
	if access == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not_authenticated"})
		return
	}

	var req changePasswordInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OldPassword = strings.TrimSpace(req.OldPassword)
	req.NewPassword = strings.TrimSpace(req.NewPassword)
	if req.OldPassword == "" || req.NewPassword == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "old_and_new_password_required"})
		return
	}

	if _, err := srv.Cognito.ChangePassword(ctx, &cognitoidentityprovider.ChangePasswordInput{
		AccessToken:      aws.String(access),
		PreviousPassword: aws.String(req.OldPassword),
		ProposedPassword: aws.String(req.NewPassword),
	}); err != nil {
		writeAuthError(w, r, http.StatusBadRequest, "change_password_failed", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (srv Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	out, err := srv.Cognito.ListUsers(ctx, &cognitoidentityprovider.ListUsersInput{UserPoolId: aws.String(srv.UserPoolID), Limit: aws.Int32(60)})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "list_failed"})
		return
	}

	users := make([]model.User, 0, len(out.Users))
	for _, u := range out.Users {
		users = append(users, mapUser(u.Username, u.Attributes, u.UserCreateDate))
	}
	writeJSON(w, http.StatusOK, users)
}

func (srv Server) handleAdminGetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := strings.TrimSpace(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/users/"), "/"))
	if id == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	username := id
	derived := derivedUsernameFromIdentifier(id)

	out, err := srv.Cognito.AdminGetUser(ctx, &cognitoidentityprovider.AdminGetUserInput{UserPoolId: aws.String(srv.UserPoolID), Username: aws.String(username)})
	if err != nil && derived != "" && shouldTryDerivedUsername(err) {
		username = derived
		out, err = srv.Cognito.AdminGetUser(ctx, &cognitoidentityprovider.AdminGetUserInput{UserPoolId: aws.String(srv.UserPoolID), Username: aws.String(username)})
	}
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	user := mapAdminUser(out)
	writeJSON(w, http.StatusOK, user)
}

func (srv Server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req createUserInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Password = strings.TrimSpace(req.Password)
	req.UserType = normalizeUserType(req.UserType)
	if req.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email_required"})
		return
	}
	if req.UserType == "" {
		req.UserType = "free"
	}
	if !isAllowedUserType(req.UserType, true) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_user_type"})
		return
	}

	attrs := []types.AttributeType{{Name: aws.String("email"), Value: aws.String(req.Email)}}
	if req.UserType != "" {
		attrs = append(attrs, types.AttributeType{Name: aws.String(cognitoUserTypeAttr), Value: aws.String(req.UserType)})
	}

	username := derivedUsernameFromEmail(req.Email)

	createOut, err := srv.Cognito.AdminCreateUser(ctx, &cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId:     aws.String(srv.UserPoolID),
		Username:       aws.String(username),
		UserAttributes: attrs,
		MessageAction:  types.MessageActionTypeSuppress,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "create_failed"})
		return
	}

	if req.Password != "" {
		_, err = srv.Cognito.AdminSetUserPassword(ctx, &cognitoidentityprovider.AdminSetUserPasswordInput{
			UserPoolId: aws.String(srv.UserPoolID),
			Username:   aws.String(username),
			Password:   aws.String(req.Password),
			Permanent:  true,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "set_password_failed"})
			return
		}
	}

	user := mapUser(createOut.User.Username, createOut.User.Attributes, createOut.User.UserCreateDate)
	writeJSON(w, http.StatusCreated, user)
}

func (srv Server) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := strings.TrimSpace(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/users/"), "/"))
	if id == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}

	var req updateUserInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	username := id
	derived := derivedUsernameFromIdentifier(id)
	try := func(fn func(user string) error) error {
		err := fn(username)
		if err == nil {
			return nil
		}
		if derived != "" && derived != username && shouldTryDerivedUsername(err) {
			if err2 := fn(derived); err2 == nil {
				username = derived
				return nil
			} else {
				return err2
			}
		}
		return err
	}

	attrs := make([]types.AttributeType, 0, 2)
	if req.Email != nil {
		v := strings.TrimSpace(strings.ToLower(*req.Email))
		req.Email = &v
		if v == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email_required"})
			return
		}
		attrs = append(attrs, types.AttributeType{Name: aws.String("email"), Value: aws.String(v)})
	}
	if req.UserType != nil {
		v := normalizeUserType(*req.UserType)
		req.UserType = &v
		if v == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_user_type"})
			return
		}
		if !isAllowedUserType(v, true) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_user_type"})
			return
		}
		attrs = append(attrs, types.AttributeType{Name: aws.String(cognitoUserTypeAttr), Value: aws.String(v)})
	}
	if len(attrs) > 0 {
		if err := try(func(user string) error {
			_, err := srv.Cognito.AdminUpdateUserAttributes(ctx, &cognitoidentityprovider.AdminUpdateUserAttributesInput{
				UserPoolId:     aws.String(srv.UserPoolID),
				Username:       aws.String(user),
				UserAttributes: attrs,
			})
			return err
		}); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "update_failed"})
			return
		}
	}

	if req.Password != nil {
		pwd := strings.TrimSpace(*req.Password)
		if err := try(func(user string) error {
			_, err := srv.Cognito.AdminSetUserPassword(ctx, &cognitoidentityprovider.AdminSetUserPasswordInput{
				UserPoolId: aws.String(srv.UserPoolID),
				Username:   aws.String(user),
				Password:   aws.String(pwd),
				Permanent:  true,
			})
			return err
		}); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "set_password_failed"})
			return
		}
	}

	if req.Disabled != nil {
		if *req.Disabled {
			if err := try(func(user string) error {
				_, err := srv.Cognito.AdminDisableUser(ctx, &cognitoidentityprovider.AdminDisableUserInput{UserPoolId: aws.String(srv.UserPoolID), Username: aws.String(user)})
				return err
			}); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "disable_failed"})
				return
			}
		} else {
			if err := try(func(user string) error {
				_, err := srv.Cognito.AdminEnableUser(ctx, &cognitoidentityprovider.AdminEnableUserInput{UserPoolId: aws.String(srv.UserPoolID), Username: aws.String(user)})
				return err
			}); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "enable_failed"})
				return
			}
		}
	}

	var out *cognitoidentityprovider.AdminGetUserOutput
	if err := try(func(user string) error {
		o, err := srv.Cognito.AdminGetUser(ctx, &cognitoidentityprovider.AdminGetUserInput{UserPoolId: aws.String(srv.UserPoolID), Username: aws.String(user)})
		if err == nil {
			out = o
		}
		return err
	}); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	writeJSON(w, http.StatusOK, mapAdminUser(out))
}

func (srv Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := strings.TrimSpace(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/users/"), "/"))
	if id == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	username := id
	derived := derivedUsernameFromIdentifier(id)

	_, err := srv.Cognito.AdminDeleteUser(ctx, &cognitoidentityprovider.AdminDeleteUserInput{UserPoolId: aws.String(srv.UserPoolID), Username: aws.String(username)})
	if err != nil && derived != "" && shouldTryDerivedUsername(err) {
		username = derived
		_, err = srv.Cognito.AdminDeleteUser(ctx, &cognitoidentityprovider.AdminDeleteUserInput{UserPoolId: aws.String(srv.UserPoolID), Username: aws.String(username)})
	}
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func mapAdminUser(out *cognitoidentityprovider.AdminGetUserOutput) model.User {
	user := model.User{ID: aws.ToString(out.Username)}
	for _, a := range out.UserAttributes {
		switch aws.ToString(a.Name) {
		case "email":
			user.Email = aws.ToString(a.Value)
		case cognitoUserTypeAttr:
			user.UserType = aws.ToString(a.Value)
		}
	}
	user.CreatedAt = timeOrZero(out.UserCreateDate)
	return user.NormalizeForResponse()
}

func mapUser(username *string, attrs []types.AttributeType, createdAt *time.Time) model.User {
	u := model.User{ID: aws.ToString(username)}
	for _, a := range attrs {
		switch aws.ToString(a.Name) {
		case "email":
			u.Email = aws.ToString(a.Value)
		case cognitoUserTypeAttr:
			u.UserType = aws.ToString(a.Value)
		case cognitoEntitlementsAttr:
			u.Entitlements = aws.ToString(a.Value)
		}
	}
	u.CreatedAt = timeOrZero(createdAt)
	return u.NormalizeForResponse()
}

func timeOrZero(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func getUserFromAccessToken(ctx context.Context, api cognito.API, accessToken string) (model.User, error) {
	if accessToken == "" {
		return model.User{}, errors.New("missing token")
	}
	out, err := api.GetUser(ctx, &cognitoidentityprovider.GetUserInput{AccessToken: aws.String(accessToken)})
	if err != nil {
		return model.User{}, err
	}
	user := model.User{ID: aws.ToString(out.Username)}
	for _, a := range out.UserAttributes {
		switch aws.ToString(a.Name) {
		case "email":
			user.Email = aws.ToString(a.Value)
		case cognitoUserTypeAttr:
			user.UserType = aws.ToString(a.Value)
		case cognitoEntitlementsAttr:
			user.Entitlements = aws.ToString(a.Value)
		}
	}
	return user, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func setCookie(w http.ResponseWriter, name, value string, secure bool, sameSite http.SameSite) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   int((24 * time.Hour).Seconds()),
	})
}

func clearCookie(w http.ResponseWriter, name string, secure bool, sameSite http.SameSite) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   -1,
	})
}

func readCookie(r *http.Request, name string) (string, bool) {
	c, err := r.Cookie(name)
	if err != nil {
		return "", false
	}
	if c.Value == "" {
		return "", false
	}
	return c.Value, true
}
