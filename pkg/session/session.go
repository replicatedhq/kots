package session

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kotsadm/pkg/user"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

type Session struct {
	ID        string
	CreatedAt time.Time
	ExpiresAt time.Time
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func Create(forUser *user.User) (*Session, error) {
	logger.Debug("creating session")

	randomID, err := ksuid.NewRandom()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate random session id")
	}

	id := randomID.String()

	db := persistence.MustGetPGSession()
	query := `insert into session (id, user_id, metadata, expire_at) values ($1, $2, $3, $4)`
	_, err = db.Exec(query, id, forUser.ID, "", time.Now().AddDate(0, 0, 14))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create session")
	}

	return get(id)
}

func Parse(signedToken string) (*Session, error) {
	if signedToken == "" {
		return nil, errors.New("missing token")
	}
	tokenParts := strings.Split(signedToken, " ")
	if len(tokenParts) != 2 {
		return nil, errors.New("invalid number of components in authorization header")
	}
	if tokenParts[0] != "Bearer" {
		return nil, errors.New("expected bearer token")
	}

	token, err := jwt.Parse(tokenParts[1], func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(os.Getenv("SESSION_KEY")), nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to parse jwt token")
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return get(claims["sessionId"].(string))
	}

	return nil, errors.New("not a valid jwttoken")
}

func get(id string) (*Session, error) {
	logger.Debug("getting session from database",
		zap.String("id", id))

	db := persistence.MustGetPGSession()
	query := `select id, expire_at from session where id = $1`
	row := db.QueryRow(query, id)
	session := Session{}

	var expiresAt time.Time
	if err := row.Scan(&session.ID, &expiresAt); err != nil {
		return nil, errors.Wrap(err, "failed to get session")
	}

	session.ExpiresAt = expiresAt

	return &session, nil
}

func (s Session) SignJWT() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sessionId": s.ID,
	})
	signedToken, err := token.SignedString([]byte(os.Getenv("SESSION_KEY")))
	if err != nil {
		return "", errors.Wrap(err, "failed to sign jwt")
	}

	return signedToken, nil
}
