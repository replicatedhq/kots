package main

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/soheilhy/cmux"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	certutil "k8s.io/client-go/util/cert"
)

var (
	TLSCiperSuites = []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	}
)

type cert struct {
	tlsCert                tls.Certificate
	fingerprint            string
	acceptAnonymousUploads bool
}

func main() {
	log.Printf("Commit %s\n", os.Getenv("COMMIT"))

	upstreamOrigin := os.Getenv("UPSTREAM_ORIGIN")
	dexUpstreamOrigin := os.Getenv("DEX_UPSTREAM_ORIGIN")
	tlsSecretName := os.Getenv("TLS_SECRET_NAME")
	namespace := os.Getenv("NAMESPACE")
	nodePort := os.Getenv("NODE_PORT")
	if nodePort == "" {
		nodePort = "8800"
	}

	gin.SetMode(gin.ReleaseMode)

	upstream, err := url.Parse(upstreamOrigin)
	if err != nil {
		log.Panic(err)
	}
	var dexUpstream *url.URL
	if dexUpstreamOrigin != "" {
		u, err := url.Parse(dexUpstreamOrigin)
		if err != nil {
			log.Panic(err)
		}
		dexUpstream = u
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panic(err)
	}
	// TODO: Assert namespace not empty, else we get secrets from all namespaces
	secrets := clientset.CoreV1().Secrets(namespace)

	_, err = secrets.Get(context.Background(), tlsSecretName, metav1.GetOptions{})
	if err != nil {
		log.Print("creating default tls secret")

		err = generateDefaultCertSecret(secrets, namespace)
		if err != nil {
			log.Printf("Could not regenerate default certificate: %v", err)
		}
		// TODO: Why are we exiting here? It leads to a pod restart
		return
	}

	certs := make(chan cert)
	go watchSecret(certs, tlsSecretName, secrets)

	var httpServer *http.Server
	var httpsServer *http.Server
	var listener net.Listener

	log.Printf("Waiting for TLS credentials from secret %s", tlsSecretName)
	for cert := range certs {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		if httpServer != nil {
			httpServer.Shutdown(ctx)
		}
		if httpsServer != nil {
			httpsServer.Shutdown(ctx)
		}
		cancel()
		if listener != nil {
			listener.Close()
		}

		l, err := net.Listen("tcp", ":"+nodePort)
		if err != nil {
			log.Panic(err)
		}
		listener = l

		m := cmux.New(listener)

		assetsDir := "/assets"

		httpsServer = getHttpsServer(upstream, dexUpstream, tlsSecretName, secrets, cert.acceptAnonymousUploads, assetsDir)
		tlsConfig := &tls.Config{
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
			CipherSuites:             TLSCiperSuites,
			Certificates:             []tls.Certificate{cert.tlsCert},
		}
		go httpsServer.Serve(tls.NewListener(m.Match(cmux.TLS()), tlsConfig))

		httpServer = getHttpServer(cert.fingerprint, cert.acceptAnonymousUploads, assetsDir)
		go httpServer.Serve(m.Match(cmux.Any()))

		log.Printf("Kurl Proxy listening on :%s\n", nodePort)
		log.Printf("\tupstream: %s\n", upstreamOrigin)
		if dexUpstream != nil {
			log.Printf("\tdex upstream: %s\n", dexUpstreamOrigin)
		}
		log.Printf("\tcert: %s\n", cert.fingerprint)
		log.Printf("\tanonymous uploads enabled: %t\n", cert.acceptAnonymousUploads)

		go func() {
			err := m.Serve()
			log.Printf("Cmux server terminated with %v", err)
		}()
	}
}

func watchSecret(certs chan cert, name string, secrets corev1.SecretInterface) {
	opts := metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", name).String(),
	}
	for {
		w, err := secrets.Watch(context.TODO(), opts)
		if err != nil {
			log.Printf("Failed to watch secret %s: %v", name, err)
			time.Sleep(time.Second * 5)
			continue
		}
		log.Printf("Watching for changes to secret %s", name)
		for e := range w.ResultChan() {
			switch e.Type {
			case watch.Added:
				fallthrough
			case watch.Modified:
				secret, ok := e.Object.(*v1.Secret)
				if !ok {
					log.Printf("Watched object wasn't a secret")
					break
				}
				certData := secret.Data["tls.crt"]
				keyData := secret.Data["tls.key"]
				crt, err := tls.X509KeyPair(certData, keyData)
				if err != nil {
					log.Printf("Ignoring secret %s: invalid cert/key pair: %v", name, err)
					break
				}

				fingerprint, err := getFingerprint(certData)
				if err != nil {
					log.Printf("Ignoring secret %s: %v", name, err)
					break
				}

				acceptAnonymousUploads := false
				if secret.Type == "Opaque" {
					// Old version of secret was type 'Opaque' and
					// the flag was stored in the Data field.  The new flag is stored as
					// an annotation.
					acceptAnonymousUploadsVal, ok := secret.Data["acceptAnonymousUploads"]
					if ok && string(acceptAnonymousUploadsVal) == "1" {
						acceptAnonymousUploads = true
					}
				} else {
					acceptAnonymousUploadsVal, ok := secret.Annotations["acceptAnonymousUploads"]
					if ok && acceptAnonymousUploadsVal == "1" {
						acceptAnonymousUploads = true
					}
				}

				certs <- cert{
					tlsCert:                crt,
					fingerprint:            fingerprint,
					acceptAnonymousUploads: acceptAnonymousUploads,
				}
			}
		}
		log.Printf("Watch of secret %s unexpectedly terminated. Reconnecting...\n", name)
		time.Sleep(time.Second * 5)
	}
}

func getFingerprint(certData []byte) (string, error) {
	var derBlock *pem.Block
	for {
		derBlock, certData = pem.Decode(certData)
		if derBlock == nil {
			return "", errors.New("no PEM data found in certificate")
		}
		if derBlock.Type == "CERTIFICATE" {
			break
		}
		if len(certData) == 0 {
			return "", errors.New("no PEM data of type CERTIFICATE found in certificate")
		}
	}

	x509Cert, err := x509.ParseCertificate(derBlock.Bytes)
	if err != nil {
		return "", err
	}
	//sha1 fingerprint is the hash of the certificate in DER form
	return strings.ToUpper(strings.Replace(fmt.Sprintf("% x", sha1.Sum(x509Cert.Raw)), " ", ":", -1)), nil
}

func getHttpServer(fingerprint string, acceptAnonymousUploads bool, assetsDir string) *http.Server {
	r := gin.Default()

	r.Use(CSPMiddleware)

	r.StaticFS("/assets", http.Dir(assetsDir))
	r.LoadHTMLGlob(fmt.Sprintf("%s/*.html", assetsDir))

	r.GET("/", func(c *gin.Context) {
		if !acceptAnonymousUploads {
			log.Println("TLS certs already uploaded, redirecting to https")
			target := url.URL{
				Scheme:   "https",
				Host:     c.Request.Host,
				Path:     c.Request.URL.Path,
				RawQuery: c.Request.URL.RawQuery,
			}
			// Returns StatusFound (302) to avoid browser caching
			c.Redirect(http.StatusFound, target.String())
			return
		}

		app, err := kotsadmApplication()

		if err != nil {
			log.Printf("No kotsadm application metadata: %v", err) // continue
		}
		appIcon := template.URL(app.Spec.Icon)
		c.HTML(http.StatusOK, "insecure.html", gin.H{
			"fingerprintSHA1": fingerprint,
			"AppIcon":         appIcon,
			"AppTitle":        app.Spec.Title,
		})
	})
	r.NoRoute(func(c *gin.Context) {
		target := url.URL{
			Scheme:   "https",
			Host:     c.Request.Host,
			Path:     c.Request.URL.Path,
			RawQuery: c.Request.URL.RawQuery,
		}
		c.Redirect(http.StatusMovedPermanently, target.String())
	})

	return &http.Server{
		Handler: r,
	}
}

func getHttpsServer(upstream, dexUpstream *url.URL, tlsSecretName string, secrets corev1.SecretInterface, acceptAnonymousUploads bool, assetsDir string) *http.Server {
	r := gin.Default()

	r.Use(CSPMiddleware)

	r.StaticFS("/tls/assets", http.Dir(assetsDir))
	r.LoadHTMLGlob(fmt.Sprintf("%s/*.html", assetsDir))

	r.GET("/tls", func(c *gin.Context) {
		if !acceptAnonymousUploads {
			c.AbortWithStatus(403)
			return
		}

		app, err := kotsadmApplication()

		if err != nil {
			log.Printf("No kotsadm application metadata: %v", err) // continue
		}
		appIcon := template.URL(app.Spec.Icon)
		c.HTML(http.StatusOK, "tls.html", gin.H{
			"Secret":   tlsSecretName,
			"AppIcon":  appIcon,
			"AppTitle": app.Spec.Title,
		})
	})

	r.POST("/tls/skip", func(c *gin.Context) {
		if !acceptAnonymousUploads {
			c.AbortWithStatus(403)
			return
		}

		hostString, success := c.GetPostForm("hostname")
		if !success {
			log.Println("Invalid hostname")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		log.Printf("Skipping TLS cert upload. Generating self-signed cert for %q", hostString)

		secret, err := secrets.Get(c.Request.Context(), tlsSecretName, metav1.GetOptions{})
		if err != nil {
			log.Print(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		go func() {
			<-c.Request.Context().Done()

			if len(secret.StringData) == 0 {
				secret.StringData = make(map[string]string)
			}
			if hostString != "" && hostString != string(secret.Data["hostname"]) {
				secret.StringData["hostname"] = hostString

				certData, keyData, err := regenerateCertWithHostname(secret.Data["tls.crt"], hostString)
				if err != nil {
					log.Println(errors.Wrapf(err, "failed to generate self-signed cert"))
				} else if certData != nil && keyData != nil {
					secret.Data["tls.crt"] = certData
					secret.Data["tls.key"] = keyData
				}
			}

			if secret.Type == "Opaque" {
				// Old version version of secret was type 'Opaque'
				delete(secret.Data, "acceptAnonymousUploads")
			} else {
				delete(secret.Annotations, "acceptAnonymousUploads")
			}
			_, err = secrets.Update(context.Background(), secret, metav1.UpdateOptions{})
			if err != nil {
				log.Printf("POST /tls/skip: %v", err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}()
	})

	r.GET("/tls/meta", func(c *gin.Context) {
		data := map[string]interface{}{
			"acceptAnonymousUploads": acceptAnonymousUploads,
		}
		c.JSON(http.StatusOK, data)
	})

	r.POST("/tls", func(c *gin.Context) {
		if !acceptAnonymousUploads {
			c.AbortWithStatus(403)
			return
		}

		certData, keyData, err := getUploadedCerts(c)
		if err != nil {
			log.Printf("POST /tls: %v", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		hostString, success := c.GetPostForm("hostname")
		if !success {
			log.Println("Invalid hostname")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		log.Printf("hostname=%v", hostString)

		err = validateCerts(certData, keyData, hostString)
		if err != nil {
			log.Printf("POST /tls: %v", err)
			data := map[string]interface{}{
				// TODO we're still using Go v1.12
				// Go v1.13 has Unwrap() and this would reduce to:
				// "error": errors.Unwrap(err),
				"error": errors.Cause(err).Error(),
			}
			c.JSON(http.StatusBadRequest, data)
			return
		}

		secret, err := secrets.Get(c.Request.Context(), tlsSecretName, metav1.GetOptions{})
		if err != nil {
			log.Print(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		go func() {
			<-c.Request.Context().Done()
			secret.Data["tls.crt"] = certData
			secret.Data["tls.key"] = keyData

			if len(secret.StringData) == 0 {
				secret.StringData = make(map[string]string)
			}
			if hostString != "" {
				secret.StringData["hostname"] = hostString
			}

			if secret.Type == "Opaque" {
				// Old version version of secret was type 'Opaque'
				delete(secret.Data, "acceptAnonymousUploads")
			} else {
				delete(secret.Annotations, "acceptAnonymousUploads")
			}
			_, err = secrets.Update(context.Background(), secret, metav1.UpdateOptions{})
			if err != nil {
				log.Print(err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}()
	})

	// these paths should not be exposed outside the cluster
	r.GET("/license/v1/license", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusForbidden)
	})
	r.POST("/api/v1/app/custom-metrics", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusForbidden)
	})

	if dexUpstream != nil {
		r.Any("/dex/*path", gin.WrapH(httputil.NewSingleHostReverseProxy(dexUpstream)))
	}

	r.NoRoute(gin.WrapH(httputil.NewSingleHostReverseProxy(upstream)))

	return &http.Server{
		Handler: r,
	}
}

// CSPMiddleware adds Content-Security-Policy and X-Frame-Options headers to the response.
func CSPMiddleware(c *gin.Context) {
	c.Writer.Header().Set("Content-Security-Policy", "frame-ancestors 'none';")
	c.Writer.Header().Set("X-Frame-Options", "DENY")
	c.Next()
}

func getUploadedCerts(c *gin.Context) ([]byte, []byte, error) {
	certHeader, err := c.FormFile("cert")
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get cert file")
	}
	certFile, err := certHeader.Open()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "open cert file")
	}
	defer certFile.Close()
	certData, err := io.ReadAll(certFile)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "read cert file")
	}

	keyHeader, err := c.FormFile("key")
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get key file")
	}
	keyFile, err := keyHeader.Open()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "open key file")
	}
	defer keyFile.Close()
	keyData, err := io.ReadAll(keyFile)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "read key file")
	}

	return certData, keyData, nil
}

func validateCerts(certData []byte, keyData []byte, hostString string) error {
	// Validates if Cert & Key match
	c, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return errors.Wrapf(err, "Cert/key pair verification failed")
	}

	// Validates cert expiration
	cert, err := x509.ParseCertificate(c.Certificate[0])
	if err != nil {
		return errors.Wrapf(err, "parsing cert/key pair")
	}
	startdate := cert.NotBefore
	enddate := cert.NotAfter
	now := time.Now()
	log.Printf("x509 cert expirations: \nstart=%v\nend=%v\nnow=%v", startdate, enddate, now)
	if now.Before(startdate) || now.After(enddate) {
		return errors.New("Certificate expired")
	}

	// Validates hostname matches cert (if hostname was specified)
	if len(hostString) > 0 {
		err := cert.VerifyHostname(hostString)
		if err != nil {
			return errors.Wrapf(err, "Hostname verification failed")
		}
	}

	return nil
}

type kotsadmAppSpec struct {
	Title string `yaml:"title"`
	Icon  string `yaml:"icon"`
}
type kotsadmApp struct {
	Spec kotsadmAppSpec `yaml:"spec"`
}

func kotsadmApplication() (kotsadmApp, error) {
	app := kotsadmApp{}

	data, err := os.ReadFile("/etc/kotsadm/application.yaml")
	if err != nil {
		return app, errors.Wrap(err, "read file /etc/kotsadm/application.yaml")
	}
	err = yaml.Unmarshal(data, &app)
	if err != nil {
		return app, errors.Wrap(err, "unmarshal /etc/kotsadm/application.yaml")
	}

	return app, nil
}

func regenerateCertWithHostname(certData []byte, hostname string) ([]byte, []byte, error) {
	log.Printf("Regenerating cert with hostname %q ...", hostname)
	certs, err := certutil.ParseCertsPEM(certData)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "parse kurl proxy secret tls.crt")
	}
	if len(certs) == 0 {
		return nil, nil, errors.Errorf("no certs found in kurl proxy secret tls.crt")
	}
	cert := certs[0]

	// Abort if current cert is a custom uploaded cert
	// If generated by kurl installer Subject will be "kotsadm.default.svc.cluster.local" and Issuer
	// will be empty. If already rotated by ekco then Subject will be like
	// "kotsadm.default.svc.cluster.local@1604697213" and Issuer like
	// "kotsadm.default.svc.cluster.local-ca@1604697213"
	if cert.Issuer.CommonName != "" && !strings.HasPrefix(cert.Issuer.CommonName, "kotsadm.default.svc.cluster.local") {
		return nil, nil, errors.New("cert issuer common name is not kotsadm.default.svc.cluster.local")
	}
	if !strings.HasPrefix(cert.Subject.CommonName, "kotsadm.default.svc.cluster.local") {
		return nil, nil, errors.New("cert common name is not kotsadm.default.svc.cluster.local")
	}

	dnsNames := cleanStringSlice(append(cert.DNSNames, hostname))

	// Generate a new self-signed cert
	certData, keyData, err := certutil.GenerateSelfSignedCertKey("kotsadm.default.svc.cluster.local", cert.IPAddresses, dnsNames)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "generate self-signed cert")
	}

	log.Printf("Cert regenerated with hostname %q", hostname)
	return certData, keyData, nil
}

func cleanStringSlice(strSlice []string) []string {
	keys := make(map[string]bool)
	clean := []string{}
	for _, entry := range strSlice {
		if entry == "" {
			continue
		}
		if _, value := keys[entry]; !value {
			keys[entry] = true
			clean = append(clean, entry)
		}
	}
	return clean
}

func generateDefaultCertSecret(secrets corev1.SecretInterface, namespace string) error {
	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-tls",
			Namespace: namespace,
			Annotations: map[string]string{
				"acceptAnonymousUploads": "1",
			},
		},
		Type:       "kubernetes.io/tls",
		Data:       make(map[string][]byte),
		StringData: make(map[string]string),
	}

	// TODO: Why is the namespace always "default"?
	// Requests to kotsadm.embeddded-cluster.svc.cluster for example will
	// fail with invalid cert errors
	altNames := []string{
		"kotsadm",
		"kotsadm.default",
		"kotsadm.default.svc",
		"kotsadm.default.svc.cluster",
		"kotsadm.default.svc.cluster.local",
	}

	hostname := "kotsadm.default.svc.cluster.local"

	// Generate a new self-signed cert
	certData, keyData, err := certutil.GenerateSelfSignedCertKey(hostname, nil, altNames)
	if err != nil {
		return errors.Wrapf(err, "generate self-signed cert")
	}

	secret.Data["tls.crt"] = certData
	secret.Data["tls.key"] = keyData
	secret.StringData["hostname"] = hostname

	_, err = secrets.Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "save tls secret")
	}

	return nil
}
