package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dekarrin/tunaq/server/dao"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func (tqs TunaQuestServer) requireJWT(ctx context.Context, req *http.Request) (dao.User, error) {
	var user dao.User

	tok, err := getJWT(req)
	if err != nil {
		return dao.User{}, err
	}

	_, err = jwt.Parse(tok, func(t *jwt.Token) (interface{}, error) {
		// who is the user? we need this for further verification
		subj, err := t.Claims.GetSubject()
		if err != nil {
			return nil, fmt.Errorf("cannot get subject: %w", err)
		}

		id, err := uuid.Parse(subj)
		if err != nil {
			return nil, fmt.Errorf("cannot parse subject UUID: %w", err)
		}

		user, err = tqs.db.Users().GetByID(ctx, id)
		if err != nil {
			if err == dao.ErrNotFound {
				return nil, fmt.Errorf("subject does not exist")
			} else {
				return nil, fmt.Errorf("subject could not be validated")
			}
		}

		var signKey []byte
		signKey = append(signKey, tqs.jwtSecret...)
		signKey = append(signKey, []byte(user.Password)...)
		signKey = append(signKey, []byte(fmt.Sprintf("%d", user.LastLogoutTime.Unix()))...)
		return signKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Alg()}), jwt.WithIssuer("tqs"), jwt.WithLeeway(time.Minute))

	if err != nil {
		return dao.User{}, err
	}

	return user, nil
}

func getJWT(req *http.Request) (string, error) {
	authHeader := strings.TrimSpace(req.Header.Get("Authorization"))

	if authHeader == "" {
		return "", fmt.Errorf("no authorization header present")
	}

	authParts := strings.SplitN(authHeader, " ", 2)
	if len(authParts) != 2 {
		return "", fmt.Errorf("authorization header not in Bearer format")
	}

	scheme := strings.TrimSpace(strings.ToLower(authParts[0]))
	token := strings.TrimSpace(authParts[1])

	if scheme != "bearer" {
		return "", fmt.Errorf("authorization header not in Bearer format")
	}

	return token, nil
}

func (tqs TunaQuestServer) generateJWT(u dao.User) (string, error) {
	claims := &jwt.MapClaims{
		"iss":        "tqs",
		"exp":        time.Now().Add(time.Hour).Unix(),
		"sub":        u.ID.String(),
		"authorized": true,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)

	var signKey []byte
	signKey = append(signKey, tqs.jwtSecret...)
	signKey = append(signKey, []byte(u.Password)...)
	signKey = append(signKey, []byte(fmt.Sprintf("%d", u.LastLogoutTime.Unix()))...)

	tokStr, err := tok.SignedString(signKey)
	if err != nil {
		return "", err
	}
	return tokStr, nil
}
