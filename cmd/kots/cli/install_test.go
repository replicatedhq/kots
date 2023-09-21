package cli_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	. "github.com/replicatedhq/kots/cmd/kots/cli"
	"github.com/replicatedhq/kots/pkg/automation"
	"github.com/replicatedhq/kots/pkg/handlers"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	preflighttypes "github.com/replicatedhq/kots/pkg/preflight/types"
	"github.com/replicatedhq/kots/pkg/store/kotsstore"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/troubleshoot/pkg/preflight"
)

var _ = Describe("Install", func() {
	Describe("ValidatePreflightStatus", func() {
		var (
			authSlug           string
			appSlug            string
			validLicense       *kotsv1beta1.License
			server             *ghttp.Server
			validDeployOptions kotsadmtypes.DeployOptions
		)

		BeforeEach(func() {
			authSlug = "test-auth-slug"
			appSlug = "test-app"

			validLicense = &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					AppSlug: "test-app",
				},
			}

			validDeployOptions = kotsadmtypes.DeployOptions{
				Namespace:         "test-namespace",
				Timeout:           time.Second,
				PreflightsTimeout: time.Second,
				License:           validLicense,
			}

			server = ghttp.NewServer()
		})

		It("returns an error when the preflight timeout deploy option is satisfied", func() {
			deployOptions := kotsadmtypes.DeployOptions{
				Namespace:         "test-namespace",
				Timeout:           time.Nanosecond,
				PreflightsTimeout: time.Nanosecond,
				License:           validLicense,
			}

			inProgressPreflightResponse, err := createPreflightResponse(false, false, true, false)
			Expect(err).ToNot(HaveOccurred())

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/app/%s/preflight/result", appSlug)),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, inProgressPreflightResponse),
				),
			)

			server.AllowUnhandledRequests = true
			err = ValidatePreflightStatus(deployOptions, authSlug, server.URL())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("timeout waiting for preflights to finish"))
		})

		It("returns an error when it cannot get a preflight response", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/app/%s/preflight/result", appSlug)),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusBadRequest, `{}`),
				),
			)

			err := ValidatePreflightStatus(validDeployOptions, authSlug, server.URL())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get preflight status"))
			Expect(err.Error()).To(ContainSubstring("unexpected status code"))
		})

		It("returns an error when there is an issue with the request", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/app/%s/preflight/result", appSlug)),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, `{}`),
				),
			)

			err := ValidatePreflightStatus(validDeployOptions, authSlug, "Invalid server URL")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to execute request"))
		})

		It("returns an error when there is an issue with the response", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/app/%s/preflight/result", appSlug)),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, `{}`, http.Header{"Content-Length": []string{"1"}}),
				),
			)

			err := ValidatePreflightStatus(validDeployOptions, authSlug, server.URL())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read response body"))
		})

		It("returns an error when the preflight response is invalid", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/app/%s/preflight/result", appSlug)),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, `{invalid json}`),
				),
			)

			err := ValidatePreflightStatus(validDeployOptions, authSlug, server.URL())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to unmarshal the preflight response"))
		})

		It("rechecks the endpoint if preflight response is empty", func() {
			longerTimeoutDeployOptions := kotsadmtypes.DeployOptions{
				Namespace:         "test-namespace",
				Timeout:           2 * time.Second,
				PreflightsTimeout: 2 * time.Second,
				License:           validLicense,
			}
			server.AllowUnhandledRequests = false

			pendingResults, err := createPreflightResponse(false, false, false, true)
			Expect(err).ToNot(HaveOccurred())

			completedPreflightResponse, err := createPreflightResponse(false, false, false, false)
			Expect(err).ToNot(HaveOccurred())

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/app/%s/preflight/result", appSlug)),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, pendingResults),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/app/%s/preflight/result", appSlug)),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, completedPreflightResponse),
				),
			)

			err = ValidatePreflightStatus(longerTimeoutDeployOptions, authSlug, server.URL())
			Expect(err).ToNot(HaveOccurred())
		})

		It("rechecks the endpoint if preflight checks are still being collected", func() {
			longerTimeoutDeployOptions := kotsadmtypes.DeployOptions{
				Namespace:         "test-namespace",
				Timeout:           2 * time.Second,
				PreflightsTimeout: 2 * time.Second,
				License:           validLicense,
			}
			server.AllowUnhandledRequests = false

			inProgressPreflightResponse, err := createPreflightResponse(false, false, true, false)
			Expect(err).ToNot(HaveOccurred())

			completedPreflightResponse, err := createPreflightResponse(false, false, false, false)
			Expect(err).ToNot(HaveOccurred())

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/app/%s/preflight/result", appSlug)),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, inProgressPreflightResponse),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/app/%s/preflight/result", appSlug)),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, completedPreflightResponse),
				),
			)

			err = ValidatePreflightStatus(longerTimeoutDeployOptions, authSlug, server.URL())
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if the preflight collection results cannot be parsed", func() {
			invalidPreflightCollection, err := json.Marshal(handlers.GetPreflightResultResponse{
				PreflightProgress: `{invalid: json}`,
				PreflightResult:   preflighttypes.PreflightResult{},
			})

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/app/%s/preflight/result", appSlug)),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, invalidPreflightCollection),
				),
			)

			err = ValidatePreflightStatus(validDeployOptions, authSlug, server.URL())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to unmarshal collect progress for preflights"))
		})

		It("returns an error if the upload preflight results are invalid", func() {
			invalidUploadPreflightResponse, err := json.Marshal(handlers.GetPreflightResultResponse{
				PreflightProgress: "",
				PreflightResult: preflighttypes.PreflightResult{
					Result: "{invalid: json}",
				},
			})
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/app/%s/preflight/result", appSlug)),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, invalidUploadPreflightResponse),
				),
			)

			err = ValidatePreflightStatus(validDeployOptions, authSlug, server.URL())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to unmarshal upload preflight results"))
		})

		DescribeTable("warning and failure preflight states", func(isFail bool, isWarn bool, expectedErr string) {
			preflightResponse, err := createPreflightResponse(isFail, isWarn, false, false)
			Expect(err).ToNot(HaveOccurred())

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/app/%s/preflight/result", appSlug)),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, preflightResponse),
				),
			)

			err = ValidatePreflightStatus(validDeployOptions, authSlug, server.URL())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(expectedErr))
		},
			Entry("warnings and failures", true, true, "Preflight checks have warnings or errors"),
			Entry("warnings only", false, true, "Preflight checks have warnings or errors"),
			Entry("failures only", true, false, "Preflight checks have warnings or errors"),
		)

		It("does not return an error if there are no warnings and failures", func() {
			validPreflightResponse, err := createPreflightResponse(false, false, false, false)
			Expect(err).ToNot(HaveOccurred())

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/app/%s/preflight/result", appSlug)),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, validPreflightResponse),
				),
			)

			err = ValidatePreflightStatus(validDeployOptions, authSlug, server.URL())
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("ValidateAutomatedInstall", func() {
		var (
			authSlug           string
			validRequestURL    string
			validLicense       *kotsv1beta1.License
			server             *ghttp.Server
			validDeployOptions kotsadmtypes.DeployOptions
		)

		BeforeEach(func() {
			authSlug = "test-auth-slug"
			appSlug := "app-slug"
			validRequestURL = fmt.Sprintf("/app/%s/automated/status", appSlug)

			validLicense = &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					AppSlug: appSlug,
				},
			}

			validDeployOptions = kotsadmtypes.DeployOptions{
				Namespace: "test-namespace",
				Timeout:   time.Second,
				License:   validLicense,
			}

			server = ghttp.NewServer()
		})

		It("returns an error when the timeout deploy option is satisfied", func() {
			deployOptions := kotsadmtypes.DeployOptions{
				Namespace: "test-namespace",
				Timeout:   time.Nanosecond,
				License:   validLicense,
			}

			runningResponse, err := createTaskStatus(automation.AutomatedInstallRunning, `{"message":"Installing app...","versionStatus":"","error":""}`)
			Expect(err).ToNot(HaveOccurred())

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", validRequestURL),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, runningResponse),
				),
			)

			server.AllowUnhandledRequests = true
			_, err = ValidateAutomatedInstall(deployOptions, authSlug, server.URL())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("timeout waiting for automated install"))
		})

		It("returns an error when it cannot get a automated install status", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", validRequestURL),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusBadRequest, `{}`),
				),
			)

			_, err := ValidateAutomatedInstall(validDeployOptions, authSlug, server.URL())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get automated install status"))
			Expect(err.Error()).To(ContainSubstring("unexpected status code"))
		})

		It("returns an error when there is an issue with the request", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", validRequestURL),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, `{}`),
				),
			)

			_, err := ValidateAutomatedInstall(validDeployOptions, authSlug, "Invalid server URL")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to execute request"))
		})

		It("returns an error when there is an issue with the response", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", validRequestURL),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, `{}`, http.Header{"Content-Length": []string{"1"}}),
				),
			)

			_, err := ValidateAutomatedInstall(validDeployOptions, authSlug, server.URL())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read response body"))
		})

		It("returns an error when the automated status response is invalid", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", validRequestURL),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, `{invalid json}`),
				),
			)

			_, err := ValidateAutomatedInstall(validDeployOptions, authSlug, server.URL())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to unmarshal task status"))
		})

		It("rechecks the endpoint if the task status is still running", func() {
			longerTimeoutDeployOptions := kotsadmtypes.DeployOptions{
				Namespace: "test-namespace",
				Timeout:   2 * time.Second,
				License:   validLicense,
			}
			server.AllowUnhandledRequests = false

			runningResponse, err := createTaskStatus(automation.AutomatedInstallRunning, `{"message":"Installing app...","versionStatus":"","error":""}`)
			Expect(err).ToNot(HaveOccurred())

			successResponse, err := createTaskStatus(automation.AutomatedInstallSuccess, `{"message":"Install complete","versionStatus":"deployed","error":""}`)
			Expect(err).ToNot(HaveOccurred())

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", validRequestURL),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, runningResponse),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", validRequestURL),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, successResponse),
				),
			)

			_, err = ValidateAutomatedInstall(longerTimeoutDeployOptions, authSlug, server.URL())
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if the task status failed", func() {
			failedResponse, err := createTaskStatus(automation.AutomatedInstallFailed, `{"message":"","versionStatus":"","error":"failed-task-error"}`)
			Expect(err).ToNot(HaveOccurred())

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", validRequestURL),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, failedResponse),
				),
			)

			_, err = ValidateAutomatedInstall(validDeployOptions, authSlug, server.URL())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed-task-error"))
		})

		It("does not return an error if the task status was successful", func() {
			success, err := createTaskStatus(automation.AutomatedInstallSuccess, `{"message":"Install complete","versionStatus":"deployed","error":""}`)
			Expect(err).ToNot(HaveOccurred())

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", validRequestURL),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{authSlug},
						"Content-Type":  []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, success),
				),
			)

			_, err = ValidateAutomatedInstall(validDeployOptions, authSlug, server.URL())
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func createPreflightResponse(isFail bool, isWarn bool, pendingCompletion bool, pendingResults bool) ([]byte, error) {
	var preflightProgress = ""
	if pendingCompletion {
		collectProgress := preflight.CollectProgress{
			CurrentName:   "collect",
			CurrentStatus: "running",
		}
		collectProgressBytes, err := json.Marshal(collectProgress)
		if err != nil {
			return nil, err
		}
		preflightProgress = string(collectProgressBytes)
	}

	uploadPreflightResult := &preflight.UploadPreflightResult{
		IsFail: isFail,
		IsWarn: isWarn,
	}

	var uploadPreflightResults = ""
	if !pendingResults {
		uploadPreflightResultsBytes, err := json.Marshal(preflight.UploadPreflightResults{
			Results: []*preflight.UploadPreflightResult{uploadPreflightResult},
			Errors:  nil,
		})
		if err != nil {
			return nil, err
		}
		uploadPreflightResults = string(uploadPreflightResultsBytes)
	}
	preflightResult := preflighttypes.PreflightResult{
		Result: uploadPreflightResults,
	}

	preflightResponse, err := json.Marshal(handlers.GetPreflightResultResponse{
		PreflightProgress: preflightProgress,
		PreflightResult:   preflightResult,
	})
	if err != nil {
		return nil, err
	}

	return preflightResponse, nil
}

func createTaskStatus(status string, message string) ([]byte, error) {
	return json.Marshal(kotsstore.TaskStatus{
		Message: message,
		Status:  status,
	})
}
