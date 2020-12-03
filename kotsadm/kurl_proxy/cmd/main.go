package main

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/docker/go-connections/tlsconfig"
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
)

type cert struct {
	tlsCert                tls.Certificate
	fingerprint            string
	acceptAnonymousUploads bool
}

func main() {
	log.Printf("Commit %s\n", os.Getenv("COMMIT"))

	rand.Seed(time.Now().UnixNano())

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
	secrets := clientset.CoreV1().Secrets(namespace)

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

		httpsServer = getHttpsServer(upstream, dexUpstream, tlsSecretName, secrets, cert.acceptAnonymousUploads)
		tlsConfig := tlsconfig.ServerDefault()
		tlsConfig.Certificates = []tls.Certificate{cert.tlsCert}
		go httpsServer.Serve(tls.NewListener(m.Match(cmux.TLS()), tlsConfig))

		httpServer = getHttpServer(cert.fingerprint, cert.acceptAnonymousUploads)
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
		w, err := secrets.Watch(opts)
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
	derBlock, _ := pem.Decode(certData)
	if derBlock == nil {
		return "", errors.New("no PEM data found in certificate")
	}
	x509Cert, err := x509.ParseCertificate(derBlock.Bytes)
	if err != nil {
		return "", err
	}
	//sha1 fingerprint is the hash of the certificate in DER form
	return strings.ToUpper(strings.Replace(fmt.Sprintf("% x", sha1.Sum(x509Cert.Raw)), " ", ":", -1)), nil
}

func getHttpServer(fingerprint string, acceptAnonymousUploads bool) *http.Server {
	r := gin.Default()

	r.StaticFS("/assets", http.Dir("/assets"))
	r.LoadHTMLGlob("/assets/*.html")

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
		c.HTML(http.StatusOK, "insecure.html", gin.H{
			"fingerprintSHA1": fingerprint,
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

func getHttpsServer(upstream, dexUpstream *url.URL, tlsSecretName string, secrets corev1.SecretInterface, acceptAnonymousUploads bool) *http.Server {
	mux := http.NewServeMux()

	r := gin.Default()

	mux.Handle("/tls/assets/", http.StripPrefix("/tls/assets/", http.FileServer(http.Dir("/assets"))))
	r.LoadHTMLGlob("/assets/*.html")

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
		if success != true {
			log.Println("Invalid hostname")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		log.Printf("hostname=%v", hostString)

		secret, err := secrets.Get(tlsSecretName, metav1.GetOptions{})
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
			if hostString != "" {
				secret.StringData["hostname"] = hostString
			}

			if secret.Type == "Opaque" {
				// Old version version of secret was type 'Opaque'
				delete(secret.Data, "acceptAnonymousUploads")
			} else {
				delete(secret.Annotations, "acceptAnonymousUploads")
			}
			_, err = secrets.Update(secret)
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
		if success != true {
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

		secret, err := secrets.Get(tlsSecretName, metav1.GetOptions{})
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
			_, err = secrets.Update(secret)
			if err != nil {
				log.Print(err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}()
	})
	mux.Handle("/tls", r)
	mux.Handle("/tls/", r)

	// mux.Handle("/api/v1/kots/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	log.Println("Kots REST API not proxied.")
	// 	http.Error(w, "Not found", http.StatusNotFound)
	// }))

	if dexUpstream != nil {
		dexReverseProxy := httputil.NewSingleHostReverseProxy(dexUpstream)
		mux.Handle("/dex", dexReverseProxy)
		mux.Handle("/dex/", dexReverseProxy)
	}
	mux.Handle("/", httputil.NewSingleHostReverseProxy(upstream))

	return &http.Server{
		Handler: mux,
	}
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
	certData, err := ioutil.ReadAll(certFile)
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
	keyData, err := ioutil.ReadAll(keyFile)
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

	data, err := ioutil.ReadFile("/etc/kotsadm/application.yaml")
	if err != nil {
		return app, errors.Wrap(err, "read file /etc/kotsadm/application.yaml")
	}
	err = yaml.Unmarshal(data, &app)
	if err != nil {
		return app, errors.Wrap(err, "unmarshal /etc/kotsadm/application.yaml")
	}

	return app, nil
}
