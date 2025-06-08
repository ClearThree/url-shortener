// Package middlewares contains all the custom middlewares that are used in the project.
package middlewares

import (
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/clearthree/url-shortener/internal/app/logger"
)

// Constants used for the authorization purposes.
const (
	AuthCookieName   = "auth"      // The name of the cookie to store an auth-token.
	UserIDHeaderName = "x-user-id" // The name of the header to store the decoded userID from the token.
)

// Errors that might occur in the Auth middleware.
var (
	ErrWrongAlgorithm  = errors.New("unexpected signing method")
	ErrTokenIsNotValid = errors.New("invalid token passed")
)

type claims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
}

// GenerateJWTString generates the JWT token for the given userID.
// Might generate the userID itself, if not passed from above.
func GenerateJWTString(userID string) (string, string, error) {
	if userID == "" {
		userID = uuid.New().String()
	}
	issueTime := time.Now()
	expireTime := issueTime.Add(time.Hour * time.Duration(config.Settings.JWTExpireHours))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "clearthree",
			IssuedAt:  jwt.NewNumericDate(issueTime),
			ExpiresAt: jwt.NewNumericDate(expireTime),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(config.Settings.SecretKey))
	if err != nil {
		return "", "", err
	}
	return tokenString, userID, nil
}

// GetUserID returns the userID, extracted from the token passed as an input.
// If not valid, returns the corresponding error.
func GetUserID(tokenString string) (string, error) {
	claimsObj := &claims{}
	token, err := jwt.ParseWithClaims(tokenString, claimsObj,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				logger.Log.Warnf("unexpected signing method: %v", t.Header["alg"])
				return nil, ErrWrongAlgorithm
			}
			return []byte(config.Settings.SecretKey), nil
		})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return claimsObj.UserID, err
		}
		return "", err
	}

	if !token.Valid {
		logger.Log.Info("Token is not valid")
		return "", ErrTokenIsNotValid
	}

	return claimsObj.UserID, nil
}

// AuthMiddleware is the middleware function itself, that tries to extract the token from the request cookies,
// authorizes it and saves the userID to request headers. If not passed, generates one in advance.
func AuthMiddleware(next http.Handler) http.Handler {
	fn := func(writer http.ResponseWriter, request *http.Request) {
		token, err := request.Cookie(AuthCookieName)
		if err != nil {
			if !errors.Is(err, http.ErrNoCookie) {
				logger.Log.Error(err)
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}

			JWTString, userID, genErr := GenerateJWTString("")
			if genErr != nil {
				http.Error(writer, genErr.Error(), http.StatusInternalServerError)
				return
			}
			request.Header.Set("x-user-id", userID)
			http.SetCookie(writer, &http.Cookie{
				Name:  AuthCookieName,
				Value: JWTString,
				Path:  "/",
			})
		} else {
			userID, tokenErr := GetUserID(token.Value)
			switch {
			case errors.Is(tokenErr, ErrTokenIsNotValid), errors.Is(tokenErr, ErrWrongAlgorithm):
				userID = ""
				logger.Log.Warnf("Token is invalid: %v", tokenErr)
				fallthrough
			case errors.Is(tokenErr, jwt.ErrTokenExpired):
				JWTString, newUserID, genErr := GenerateJWTString(userID)
				if genErr != nil {
					http.Error(writer, genErr.Error(), http.StatusInternalServerError)
					return
				}
				request.Header.Set("x-user-id", newUserID)
				http.SetCookie(writer, &http.Cookie{
					Name:  AuthCookieName,
					Value: JWTString,
					Path:  "/",
				})
			case tokenErr != nil:
				logger.Log.Error(tokenErr)
				http.Error(writer, tokenErr.Error(), http.StatusInternalServerError)
				return
			}
			if userID == "" {
				http.Error(writer, "Unauthorized", http.StatusUnauthorized)
				return
			}
			request.Header.Set("x-user-id", userID)
		}

		next.ServeHTTP(writer, request)
	}
	return http.HandlerFunc(fn)
}
