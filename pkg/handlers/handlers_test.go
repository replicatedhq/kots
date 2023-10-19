package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/handlers"
	mock_handlers "github.com/replicatedhq/kots/pkg/handlers/mock"
	"github.com/replicatedhq/kots/pkg/policy"
	"github.com/replicatedhq/kots/pkg/rbac"
	rbactypes "github.com/replicatedhq/kots/pkg/rbac/types"
	"github.com/replicatedhq/kots/pkg/session"
	sessiontypes "github.com/replicatedhq/kots/pkg/session/types"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	supportbundletypes "github.com/replicatedhq/kots/pkg/supportbundle/types"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var HandlerPolicyTests = map[string][]HandlerPolicyTest{
	// Installation
	"UploadNewLicense": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.UploadNewLicense(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"ExchangePlatformLicense": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.ExchangePlatformLicense(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"ResumeInstallOnline": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.ResumeInstallOnline(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetOnlineInstallStatus": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetOnlineInstallStatus(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"CanInstallAppVersion": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.CanInstallAppVersion(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetAutomatedInstallStatus": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetAutomatedInstallStatus(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	// Support Bundles
	"GetSupportBundle": {
		{
			Vars:         map[string]string{"bundleSlug": "bundle-slug"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				storeRecorder.GetSupportBundle("bundle-slug").Return(&supportbundletypes.SupportBundle{AppID: "123"}, nil)
				storeRecorder.GetApp("123").Return(&apptypes.App{Slug: "my-app"}, nil)
				handlerRecorder.GetSupportBundle(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"ListSupportBundles": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.ListSupportBundles(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetSupportBundleCommand": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetSupportBundleCommand(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetSupportBundleFiles": {
		{
			Vars:         map[string]string{"bundleId": "234"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				storeRecorder.GetSupportBundle("234").Return(&supportbundletypes.SupportBundle{AppID: "123"}, nil)
				storeRecorder.GetApp("123").Return(&apptypes.App{Slug: "my-app"}, nil)
				handlerRecorder.GetSupportBundleFiles(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetSupportBundleRedactions": {
		{
			Vars:         map[string]string{"bundleId": "234"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				storeRecorder.GetSupportBundle("234").Return(&supportbundletypes.SupportBundle{AppID: "123"}, nil)
				storeRecorder.GetApp("123").Return(&apptypes.App{Slug: "my-app"}, nil)
				handlerRecorder.GetSupportBundleRedactions(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"DownloadSupportBundle": {
		{
			Vars:         map[string]string{"bundleId": "234"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				storeRecorder.GetSupportBundle("234").Return(&supportbundletypes.SupportBundle{AppID: "123"}, nil)
				storeRecorder.GetApp("123").Return(&apptypes.App{Slug: "my-app"}, nil)
				handlerRecorder.DownloadSupportBundle(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"CollectSupportBundle": {
		{
			Vars:         map[string]string{"appId": "123", "clusterId": "345"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				storeRecorder.GetApp("123").Return(&apptypes.App{Slug: "my-app"}, nil)
				handlerRecorder.CollectSupportBundle(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"CollectHelmSupportBundle": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.CollectHelmSupportBundle(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"ShareSupportBundle": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "bundleId": "234"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.ShareSupportBundle(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"DeleteSupportBundle": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "bundleId": "234"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.DeleteSupportBundle(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetPodDetailsFromSupportBundle": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "bundleId": "234"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetPodDetailsFromSupportBundle(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	// redactor routes
	"UpdateRedact": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.UpdateRedact(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetRedact": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetRedact(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"ListRedactors": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.ListRedactors(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetRedactMetadataAndYaml": {
		{
			Vars:         map[string]string{"slug": "redact-slug"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetRedactMetadataAndYaml(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"SetRedactMetadataAndYaml": {
		{
			Vars:         map[string]string{"slug": "redact-slug"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.SetRedactMetadataAndYaml(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"DeleteRedact": {
		{
			Vars:         map[string]string{"slug": "redact-slug"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.DeleteRedact(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"SetRedactEnabled": {
		{
			Vars:         map[string]string{"slug": "redact-slug"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.SetRedactEnabled(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	// Kotsadm Identity Service
	"ConfigureIdentityService": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.ConfigureIdentityService(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetIdentityServiceConfig": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetIdentityServiceConfig(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	// App Identity Service
	"ConfigureAppIdentityService": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.ConfigureAppIdentityService(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetAppIdentityServiceConfig": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetAppIdentityServiceConfig(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	// Apps
	"ListApps": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.ListApps(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetApp": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetApp(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetAppStatus": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetAppStatus(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetAppVersionHistory": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetAppVersionHistory(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetLatestDeployableVersion": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetLatestDeployableVersion(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetUpdateDownloadStatus": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetUpdateDownloadStatus(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	// Airgap
	"AirgapBundleProgress": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "identifier": "456", "totalChunks": "100"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.AirgapBundleProgress(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"AirgapBundleExists": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "identifier": "456", "totalChunks": "100"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.AirgapBundleExists(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"CreateAppFromAirgap": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "identifier": "456", "totalChunks": "100"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.CreateAppFromAirgap(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"UpdateAppFromAirgap": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "identifier": "456", "totalChunks": "100"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.UpdateAppFromAirgap(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"CheckAirgapBundleChunk": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.CheckAirgapBundleChunk(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"UploadAirgapBundleChunk": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.UploadAirgapBundleChunk(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetAirgapInstallStatus": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetAirgapInstallStatus(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"ResetAirgapInstallStatus": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.ResetAirgapInstallStatus(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetAirgapUploadConfig": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetAirgapUploadConfig(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	// Implemented handlers
	"IgnorePreflightRBACErrors": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "sequence": "1"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.IgnorePreflightRBACErrors(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"StartPreflightChecks": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "sequence": "1"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.StartPreflightChecks(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetLatestPreflightResultsForSequenceZero": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetLatestPreflightResultsForSequenceZero(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetPreflightResult": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "sequence": "1"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetPreflightResult(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetPreflightCommand": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "sequence": "1"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetPreflightCommand(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"PreflightsReports": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.PreflightsReports(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	"UpdateAdminConsole": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "sequence": "1"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.UpdateAdminConsole(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetAdminConsoleUpdateStatus": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetAdminConsoleUpdateStatus(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"DeployAppVersion": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "sequence": "1"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.DeployAppVersion(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"RedeployAppVersion": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "sequence": "1"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.RedeployAppVersion(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"DownloadAppVersion": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "sequence": "1"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.DownloadAppVersion(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetAppVersionDownloadStatus": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "sequence": "1"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetAppVersionDownloadStatus(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetAppRenderedContents": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "sequence": "1"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetAppRenderedContents(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetAppContents": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "sequence": "1"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetAppContents(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetAppDashboard": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "clusterId": "345"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetAppDashboard(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetDownstreamOutput": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "clusterId": "345", "sequence": "1"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetDownstreamOutput(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	"GetKotsadmRegistry": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetKotsadmRegistry(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetImageRewriteStatusOld": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetImageRewriteStatus(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	"UpdateAppRegistry": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.UpdateAppRegistry(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetAppRegistry": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetAppRegistry(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetImageRewriteStatus": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetImageRewriteStatus(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GarbageCollectImages": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GarbageCollectImages(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"DockerHubSecretUpdated": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.DockerHubSecretUpdated(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"ValidateAppRegistry": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.ValidateAppRegistry(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	"UpdateAppConfig": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.UpdateAppConfig(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"CurrentAppConfig": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "sequence": "1"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.CurrentAppConfig(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"LiveAppConfig": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.LiveAppConfig(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"SetAppConfigValues": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.SetAppConfigValues(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"DownloadFileFromConfig": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "sequence": "0", "filename": "my-file"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.DownloadFileFromConfig(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"SyncLicense": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.SyncLicense(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"ChangeLicense": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.ChangeLicense(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetLicense": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetLicense(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	"AppUpdateCheck": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.AppUpdateCheck(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"SetAutomaticUpdatesConfig": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.SetAutomaticUpdatesConfig(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetAutomaticUpdatesConfig": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetAutomaticUpdatesConfig(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"RemoveApp": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.RemoveApp(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	"CreateApplicationBackup": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.CreateApplicationBackup(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetRestoreStatus": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetRestoreStatus(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"CancelRestore": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.CancelRestore(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"CreateApplicationRestore": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "snapshotName": "snapshot-name"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.CreateApplicationRestore(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetRestoreDetails": {
		{
			Vars:         map[string]string{"appSlug": "my-app", "restoreName": "restore-name"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetRestoreDetails(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"ListBackups": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.ListBackups(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetSnapshotConfig": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetSnapshotConfig(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"SaveSnapshotConfig": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.SaveSnapshotConfig(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	"ListInstanceBackups": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.ListInstanceBackups(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"CreateInstanceBackup": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.CreateInstanceBackup(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetInstanceSnapshotConfig": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetInstanceSnapshotConfig(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"SaveInstanceSnapshotConfig": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.SaveInstanceSnapshotConfig(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetGlobalSnapshotSettings": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetGlobalSnapshotSettings(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"UpdateGlobalSnapshotSettings": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.UpdateGlobalSnapshotSettings(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetFileSystemSnapshotProviderInstructions": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetFileSystemSnapshotProviderInstructions(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetBackup": {
		{
			Vars:         map[string]string{"snapshotName": "snapshot-name"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetBackup(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"DeleteBackup": {
		{
			Vars:         map[string]string{"snapshotName": "snapshot-name"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.DeleteBackup(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"RestoreApps": {
		{
			Vars:         map[string]string{"snapshotName": "snapshot-name"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.RestoreApps(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetRestoreAppsStatus": {
		{
			Vars:         map[string]string{"snapshotName": "snapshot-name"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetRestoreAppsStatus(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"DownloadSnapshotLogs": {
		{
			Vars:         map[string]string{"backup": "backup-name"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.DownloadSnapshotLogs(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetVeleroStatus": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetVeleroStatus(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	"Kurl": {}, // Not implemented
	"GenerateKurlNodeJoinCommandWorker": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GenerateKurlNodeJoinCommandWorker(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GenerateKurlNodeJoinCommandMaster": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GenerateKurlNodeJoinCommandMaster(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GenerateKurlNodeJoinCommandSecondary": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GenerateKurlNodeJoinCommandSecondary(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GenerateKurlNodeJoinCommandPrimary": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GenerateKurlNodeJoinCommandPrimary(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"DrainKurlNode": {
		{
			Vars:         map[string]string{"nodeName": "node-name"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.DrainKurlNode(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"DeleteKurlNode": {
		{
			Vars:         map[string]string{"nodeName": "node-name"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.DeleteKurlNode(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetKurlNodes": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetKurlNodes(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	"HelmVM": {}, // Not implemented
	"GenerateHelmVMNodeJoinCommand": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GenerateHelmVMNodeJoinCommand(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"DrainHelmVMNode": {
		{
			Vars:         map[string]string{"nodeName": "node-name"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.DrainHelmVMNode(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"DeleteHelmVMNode": {
		{
			Vars:         map[string]string{"nodeName": "node-name"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.DeleteHelmVMNode(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetHelmVMNodes": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetHelmVMNodes(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetHelmVMNode": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetHelmVMNode(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetK0sNodeJoinCommand": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetK0sNodeJoinCommand(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	// Prometheus
	"SetPrometheusAddress": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.SetPrometheusAddress(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},

	// GitOps
	"UpdateAppGitOps": {
		{
			Vars:         map[string]string{"appId": "123", "clusterId": "345"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				storeRecorder.GetApp("123").Return(&apptypes.App{Slug: "my-app"}, nil)
				handlerRecorder.UpdateAppGitOps(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"DisableAppGitOps": {
		{
			Vars:         map[string]string{"appId": "123", "clusterId": "345"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				storeRecorder.GetApp("123").Return(&apptypes.App{Slug: "my-app"}, nil)
				handlerRecorder.DisableAppGitOps(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"InitGitOpsConnection": {
		{
			Vars:         map[string]string{"appId": "123", "clusterId": "345"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				storeRecorder.GetApp("123").Return(&apptypes.App{Slug: "my-app"}, nil)
				handlerRecorder.InitGitOpsConnection(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"CreateGitOps": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.CreateGitOps(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"ResetGitOps": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.ResetGitOps(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetGitOpsRepo": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetGitOpsRepo(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetPendingApp": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetPendingApp(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"ChangePassword": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.ChangePassword(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"IsHelmManaged": {
		{
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.IsHelmManaged(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"GetAppValuesFile": {
		{
			Vars:         map[string]string{"appSlug": "123", "sequence": "1"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.GetAppValuesFile(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
}

type HandlerPolicyTest struct {
	Vars         map[string]string
	Roles        []rbactypes.Role
	SessionRoles []string
	Calls        func(storeRecorder *mock_store.MockStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder)
	ExpectStatus int
}

func TestHandlerPolicies(t *testing.T) {
	r := mux.NewRouter()
	// Just enough here to walk the routes
	handlers.RegisterSessionAuthRoutes(r, nil, &handlers.Handler{}, policy.NewMiddleware(nil, nil))
	r.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		name := route.GetName()
		methods, _ := route.GetMethods()
		pathTemplate, _ := route.GetPathTemplate()
		if name == "" {
			t.Errorf("route %s %s: name required", methods, pathTemplate)
			return nil
		}

		tests, ok := HandlerPolicyTests[name]
		if !ok {
			t.Errorf("route %s: tests required", name)
			return nil
		}

		for _, method := range methods {
			for _, test := range tests {
				pairs := []string{}
				for key, val := range test.Vars {
					pairs = append(pairs, key, val)
				}
				path, err := route.URLPath(pairs...)
				require.NoError(t, err)

				t.Run(fmt.Sprintf("%s [%s] %s %d", name, method, path, test.ExpectStatus), func(t *testing.T) {
					ctrl := gomock.NewController(t)
					defer ctrl.Finish()

					kotsStoreMock := mock_store.NewMockStore(ctrl)
					kotsHandlersMock := mock_handlers.NewMockKOTSHandler(ctrl)

					middleware := policy.NewMiddleware(kotsStoreMock, test.Roles)

					r := mux.NewRouter()
					handlers.RegisterSessionAuthRoutes(r, kotsStoreMock, kotsHandlersMock, middleware)

					sess := &sessiontypes.Session{
						ID:        ksuid.New().String(),
						IssuedAt:  time.Now(),
						ExpiresAt: time.Now().Add(handlers.SessionTimeout),
						Roles:     test.SessionRoles,
						HasRBAC:   true,
					}
					signedJWT, err := session.SignJWT(sess)
					require.NoError(t, err)

					req := httptest.NewRequest(method, fmt.Sprintf("http://example.com%s", path), nil)
					req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", signedJWT))

					var match mux.RouteMatch
					if !route.Match(req, &match) {
						t.Fatal("path does not match")
					}

					kotsStoreMock.EXPECT().
						GetSession(sess.ID).
						Return(sess, nil)
					kotsStoreMock.EXPECT().
						GetPasswordUpdatedAt().
						Return(nil, nil)

					test.Calls(kotsStoreMock.EXPECT(), kotsHandlersMock.EXPECT())

					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					resp := w.Result()

					assert.Equal(t, test.ExpectStatus, resp.StatusCode)
				})
			}
		}

		return nil
	})
}

func TestListUnauthedRoutes(t *testing.T) {
	req := require.New(t)

	r := mux.NewRouter()
	handlers.RegisterUnauthenticatedRoutes(&handlers.Handler{}, nil, r, r)
	// build a list of patterns that are used by kots
	patternList := []string{}
	err := r.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err != nil {
			return err
		}
		patternList = append(patternList, path)
		return nil
	})
	req.NoError(err)

	knownPatterns := []string{
		"/healthz",
		"/api/v1/login",
		"/api/v1/login/info",
		"/api/v1/logout",
		"/api/v1/metadata",
		"/api/v1/oidc/login",
		"/api/v1/oidc/login/callback",
		"/api/v1/troubleshoot/{appId}/{bundleId}",
		"/api/v1/troubleshoot/supportbundle/{bundleId}/redactions",
		"/api/v1/preflight/app/{appSlug}/sequence/{sequence}",
		"/license/v1/license",
	}

	for _, knownPattern := range knownPatterns {
		// validate that this pattern is present within the list of unauthenticated routes and has not been removed
		found := false
		for _, currentPattern := range patternList {
			if currentPattern == knownPattern {
				found = true
			}
		}
		if !found {
			t.Errorf("api pattern %q was not found in list %v", knownPattern, patternList)
		}
	}
}

func TestListTokenAuthRoutes(t *testing.T) {
	req := require.New(t)

	r := mux.NewRouter()
	handlers.RegisterTokenAuthRoutes(&handlers.Handler{}, r, r)
	// build a list of patterns that are used by kots
	patternList := []string{}
	err := r.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err != nil {
			return err
		}
		patternList = append(patternList, path)
		return nil
	})
	req.NoError(err)

	knownPatterns := []string{
		"/api/v1/kots/ports",
		"/api/v1/upload",
		"/api/v1/download",
		"/api/v1/airgap/install",
	}

	for _, knownPattern := range knownPatterns {
		// validate that this pattern is present within the list of token auth routes and has not been removed
		found := false
		for _, currentPattern := range patternList {
			if currentPattern == knownPattern {
				found = true
			}
		}
		if !found {
			t.Errorf("api pattern %q was not found in list %v", knownPattern, patternList)
		}
	}
}
