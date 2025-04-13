package middlewares

import (
	"errors"
	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/clearthree/url-shortener/internal/app/logger"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"net/http"
	"time"
)

const AuthCookieName = "auth"
const UserIDHeaderName = "x-user-id"

var ErrWrongAlgorithm = errors.New("unexpected signing method")
var ErrTokenIsNotValid = errors.New("invalid token passed")

type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
}

func GenerateJWTString(userID string) (string, string, error) {
	if userID == "" {
		userID = uuid.New().String()
	}
	issueTime := time.Now()
	expireTime := issueTime.Add(time.Hour * time.Duration(config.Settings.JWTExpireHours))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
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

func GetUserID(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				logger.Log.Warnf("unexpected signing method: %v", t.Header["alg"])
				return nil, ErrWrongAlgorithm
			}
			return []byte(config.Settings.SecretKey), nil
		})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return claims.UserID, err
		}
		return "", err
	}

	if !token.Valid {
		logger.Log.Info("Token is not valid")
		return "", ErrTokenIsNotValid
	}

	return claims.UserID, nil
}

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
