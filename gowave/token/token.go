package token

import (
	"errors"
	"github.com/ChenGuo505/gowave"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"time"
)

type JWTHandler struct {
	Alg            string
	Expiration     time.Duration
	Refresh        time.Duration
	Key            []byte
	RefreshKey     string
	PrivateKey     string
	Authenticator  func(*gowave.Context) (map[string]any, error)
	CookieName     string
	CookieMaxAge   int64
	CookieDomain   string
	SecureCookie   bool
	CookieHTTPOnly bool
	SetCookie      bool
	Header         string
	AuthHandler    func(*gowave.Context, error)
}

type JWTResponse struct {
	Token        string
	RefreshToken string
}

func (j *JWTHandler) LoginHandler(ctx *gowave.Context) (*JWTResponse, error) {
	data, err := j.Authenticator(ctx)
	if err != nil {
		return nil, err
	}
	if j.Alg == "" {
		j.Alg = "HS256" // Default algorithm
	}
	method := jwt.GetSigningMethod(j.Alg)
	token := jwt.New(method)
	claims := token.Claims.(jwt.MapClaims)
	if data != nil {
		for k, v := range data {
			claims[k] = v
		}
	}
	now := time.Now()
	expire := now.Add(j.Expiration)
	claims["exp"] = expire.Unix()
	claims["iat"] = now.Unix()
	var tokenString string
	if j.usePublicKeyAlgorithm() {
		tokenString, err = token.SignedString(j.PrivateKey)
	} else {
		tokenString, err = token.SignedString(j.Key)
	}
	if err != nil {
		return nil, err
	}
	jr := &JWTResponse{
		Token: tokenString,
	}
	refreshToken, err := j.refreshToken(token)
	if err != nil {
		return nil, err
	}
	jr.RefreshToken = refreshToken
	if j.SetCookie {
		if j.CookieName == "" {
			j.CookieName = "gowave_toke"
		}
		if j.CookieMaxAge <= 0 {
			j.CookieMaxAge = expire.Unix() - now.Unix()
		}
		ctx.SetCookie(j.CookieName, tokenString, int(j.CookieMaxAge), "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
	}
	return jr, nil
}

func (j *JWTHandler) LogoutHandler(ctx *gowave.Context) error {
	if j.SetCookie {
		if j.CookieName == "" {
			j.CookieName = "gowave_toke"
		}
		ctx.SetCookie(j.CookieName, "", -1, "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
		return nil
	}
	return nil
}

func (j *JWTHandler) RefreshHandler(ctx *gowave.Context) (*JWTResponse, error) {
	rt, ok := ctx.Get(j.RefreshKey)
	if !ok {
		return nil, errors.New("no refresh key")
	}
	if j.Alg == "" {
		j.Alg = "HS256" // Default algorithm
	}
	t, err := jwt.Parse(rt.(string), func(token *jwt.Token) (interface{}, error) {
		if j.usePublicKeyAlgorithm() {
			return []byte(j.PrivateKey), nil
		} else {
			return j.Key, nil
		}
	})
	if err != nil {
		return nil, err
	}
	claims := t.Claims.(jwt.MapClaims)
	now := time.Now()
	expire := now.Add(j.Expiration)
	claims["exp"] = expire.Unix()
	claims["iat"] = now.Unix()
	var tokenString string
	if j.usePublicKeyAlgorithm() {
		tokenString, err = t.SignedString(j.PrivateKey)
	} else {
		tokenString, err = t.SignedString(j.Key)
	}
	if err != nil {
		return nil, err
	}
	jr := &JWTResponse{
		Token: tokenString,
	}
	refreshToken, err := j.refreshToken(t)
	if err != nil {
		return nil, err
	}
	jr.RefreshToken = refreshToken
	if j.SetCookie {
		if j.CookieName == "" {
			j.CookieName = "gowave_toke"
		}
		if j.CookieMaxAge <= 0 {
			j.CookieMaxAge = expire.Unix() - now.Unix()
		}
		ctx.SetCookie(j.CookieName, tokenString, int(j.CookieMaxAge), "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
	}
	return jr, nil
}

func (j *JWTHandler) AuthInterceptor(next gowave.HandlerFunc) gowave.HandlerFunc {
	return func(ctx *gowave.Context) {
		if j.Header == "" {
			j.Header = "Authorization"
		}
		token := ctx.Req.Header.Get(j.Header)
		if token == "" {
			if j.SetCookie {
				cookie, err := ctx.Req.Cookie(j.CookieName)
				if err != nil {
					j.authError(ctx, err)
					return
				}
				token = cookie.String()
			}
		}
		t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			if j.usePublicKeyAlgorithm() {
				return []byte(j.PrivateKey), nil
			} else {
				return j.Key, nil
			}
		})
		if err != nil {
			j.authError(ctx, err)
			return
		}
		claims := t.Claims.(jwt.MapClaims)
		ctx.Set("jwt_claims", claims)
		next(ctx)
	}
}

func (j *JWTHandler) usePublicKeyAlgorithm() bool {
	switch j.Alg {
	case "RS256", "RS384", "RS512":
		return true
	default:
		return false
	}
}

func (j *JWTHandler) refreshToken(token *jwt.Token) (string, error) {
	claims := token.Claims.(jwt.MapClaims)
	now := time.Now()
	claims["exp"] = now.Add(j.Refresh).Unix()
	var tokenString string
	var err error
	if j.usePublicKeyAlgorithm() {
		tokenString, err = token.SignedString(j.PrivateKey)
	} else {
		tokenString, err = token.SignedString(j.Key)
	}
	return tokenString, err
}

func (j *JWTHandler) authError(ctx *gowave.Context, err error) {
	if j.AuthHandler != nil {
		j.AuthHandler(ctx, err)
	} else {
		ctx.W.WriteHeader(http.StatusUnauthorized)
	}
}
