package session

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/session/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/identity"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func Parse(signedToken string) (*types.Session, error) {
	if signedToken == "" {
		return nil, errors.New("missing token")
	}
	tokenParts := strings.Split(signedToken, " ")
	if len(tokenParts) != 2 {
		return nil, errors.New("invalid number of components in authorization header")
	}
	if tokenParts[0] != "Bearer" && tokenParts[0] != "Kots" {
		return nil, errors.New("expected bearer or kots token")
	}

	if tokenParts[0] == "Kots" {
		// this is a token from the kots CLI
		// it needs to be compared with the "kotsadm-authstring" secret
		// if that matches, we return a new session token with the session ID set to the authstring value
		// and the userID set to "kots-cli"
		// this works for now as the endpoints used by the kots cli don't rely on user ID
		// TODO make real userid/sessionid
		cfg, err := config.GetConfig()
		if err != nil {
			return nil, errors.New("failed to get cluster config")
		}

		clientset, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, errors.New("failed to create clientset")
		}

		secret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), "kotsadm-authstring", metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return nil, errors.New("failed to read auth string")
		}

		if kuberneteserrors.IsNotFound(err) {
			return nil, errors.New("no auth string found")
		}

		if signedToken != string(secret.Data["kotsadm-authstring"]) {
			return nil, errors.New("invalid authstring")
		}

		s := types.Session{
			ID:        "kots-cli",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Minute),
			// TODO: super user permissions
			Roles:   GetSessionRolesFromRBAC(nil, identity.DefaultGroups),
			HasRBAC: true,
		}

		return &s, nil
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
		return store.GetStore().GetSession(claims["sessionId"].(string))
	}

	return nil, errors.New("not a valid jwttoken")
}

func SignJWT(s *types.Session) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sessionId": s.ID,
	})
	signedToken, err := token.SignedString([]byte(os.Getenv("SESSION_KEY")))
	if err != nil {
		return "", errors.Wrap(err, "failed to sign jwt")
	}

	return signedToken, nil
}

func GetSessionRolesFromRBAC(sessionGroupIDs []string, groups []kotsv1beta1.IdentityGroup) []string {
	var sessionRolesIDs []string
	for _, group := range groups {
		if group.ID == identity.WildcardGroupID {
			sessionRolesIDs = append(sessionRolesIDs, group.RoleIDs...)
			continue
		}
		for _, groupID := range sessionGroupIDs {
			if group.ID == groupID {
				sessionRolesIDs = append(sessionRolesIDs, group.RoleIDs...)
				break
			}
		}
	}
	return sessionRolesIDs
}
