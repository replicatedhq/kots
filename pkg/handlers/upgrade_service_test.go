package handlers_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/handlers"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/reporting"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	"github.com/replicatedhq/kots/pkg/update"
	"github.com/replicatedhq/kots/pkg/upgradeservice"
	upgradeservicetypes "github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestStartUpgradeService(t *testing.T) {
	// mock replicated app
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/clusterconfig/version/Installer":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"version": "online-update-ec-version"})

		case "/clusterconfig/artifact/kots":
			kotsTGZ := mockKOTSBinary(t)
			w.WriteHeader(http.StatusOK)
			w.Write(kotsTGZ)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// mock update airgap bundle
	updateAirgapBundle := mockUpdateAirgapBundle(t)
	defer os.Remove(updateAirgapBundle)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)

	t.Setenv("USE_MOCK_REPORTING", "1")
	t.Setenv("EMBEDDED_CLUSTER_VERSION", "current-ec-version")
	t.Setenv("MOCK_BEHAVIOR", "upgrade-service-cmd")

	testLicense := fmt.Sprintf(`apiVersion: kots.io/v1beta1
kind: License
metadata:
  name: testcustomer
spec:
  appSlug: my-app
  channelID: 1vusIYZLAVxMG6q760OJmRKj5i5
  channelName: My Channel
  customerName: Test Customer
  endpoint: %s
  entitlements:
    expires_at:
      description: License Expiration
      title: Expiration
      value: "2030-07-27T00:00:00Z"
      valueType: String
  isAirgapSupported: true
  isGitOpsSupported: true
  isSnapshotSupported: true
  licenseID: 1vusOokxAVp1tkRGuyxnF23PJcq
  licenseSequence: 7
  licenseType: prod
  signature: eyJsaWNlbnNlRGF0YSI6ImV5SmhjR2xXWlhKemFXOXVJam9pYTI5MGN5NXBieTkyTVdKbGRHRXhJaXdpYTJsdVpDSTZJa3hwWTJWdWMyVWlMQ0p0WlhSaFpHRjBZU0k2ZXlKdVlXMWxJam9pZEdWemRHTjFjM1J2YldWeUluMHNJbk53WldNaU9uc2liR2xqWlc1elpVbEVJam9pTVhaMWMwOXZhM2hCVm5BeGRHdFNSM1Y1ZUc1R01qTlFTbU54SWl3aWJHbGpaVzV6WlZSNWNHVWlPaUp3Y205a0lpd2lZM1Z6ZEc5dFpYSk9ZVzFsSWpvaVZHVnpkQ0JEZFhOMGIyMWxjaUlzSW1Gd2NGTnNkV2NpT2lKdGVTMWhjSEFpTENKamFHRnVibVZzU1VRaU9pSXhkblZ6U1ZsYVRFRldlRTFITm5FM05qQlBTbTFTUzJvMWFUVWlMQ0pqYUdGdWJtVnNUbUZ0WlNJNklrMTVJRU5vWVc1dVpXd2lMQ0pzYVdObGJuTmxVMlZ4ZFdWdVkyVWlPamNzSW1WdVpIQnZhVzUwSWpvaWFIUjBjSE02THk5eVpYQnNhV05oZEdWa0xtRndjQ0lzSW1WdWRHbDBiR1Z0Wlc1MGN5STZleUppYjI5c1gyWnBaV3hrSWpwN0luUnBkR3hsSWpvaVFtOXZiQ0JHYVdWc1pDSXNJblpoYkhWbElqcDBjblZsTENKMllXeDFaVlI1Y0dVaU9pSkNiMjlzWldGdUluMHNJbVY0Y0dseVpYTmZZWFFpT25zaWRHbDBiR1VpT2lKRmVIQnBjbUYwYVc5dUlpd2laR1Z6WTNKcGNIUnBiMjRpT2lKTWFXTmxibk5sSUVWNGNHbHlZWFJwYjI0aUxDSjJZV3gxWlNJNklqSXdNekF0TURjdE1qZFVNREE2TURBNk1EQmFJaXdpZG1Gc2RXVlVlWEJsSWpvaVUzUnlhVzVuSW4wc0ltaHBaR1JsYmw5bWFXVnNaQ0k2ZXlKMGFYUnNaU0k2SWtocFpHUmxiaUJHYVdWc1pDSXNJblpoYkhWbElqb2lkR2hwY3lCcGN5QnpaV055WlhRaUxDSjJZV3gxWlZSNWNHVWlPaUpUZEhKcGJtY2lMQ0pwYzBocFpHUmxiaUk2ZEhKMVpYMHNJbWx1ZEY5bWFXVnNaQ0k2ZXlKMGFYUnNaU0k2SWtsdWRDQkdhV1ZzWkNJc0luWmhiSFZsSWpveE1qTXNJblpoYkhWbFZIbHdaU0k2SWtsdWRHVm5aWElpZlN3aWMzUnlhVzVuWDJacFpXeGtJanA3SW5ScGRHeGxJam9pVTNSeWFXNW5SbWxsYkdRaUxDSjJZV3gxWlNJNkluTnBibWRzWlNCc2FXNWxJSFJsZUhRaUxDSjJZV3gxWlZSNWNHVWlPaUpUZEhKcGJtY2lmU3dpZEdWNGRGOW1hV1ZzWkNJNmV5SjBhWFJzWlNJNklsUmxlSFFnUm1sbGJHUWlMQ0oyWVd4MVpTSTZJbTExYkhScFhHNXNhVzVsWEc1MFpYaDBJaXdpZG1Gc2RXVlVlWEJsSWpvaVZHVjRkQ0o5ZlN3aWFYTkJhWEpuWVhCVGRYQndiM0owWldRaU9uUnlkV1VzSW1selIybDBUM0J6VTNWd2NHOXlkR1ZrSWpwMGNuVmxMQ0pwYzFOdVlYQnphRzkwVTNWd2NHOXlkR1ZrSWpwMGNuVmxmWDA9IiwiaW5uZXJTaWduYXR1cmUiOiJleUpzYVdObGJuTmxVMmxuYm1GMGRYSmxJam9pYUhneE1XTXZUR1ozUTNoVE5YRmtRWEJGU1hGdVRrMU9NMHBLYTJzNFZHZFhSVVpzVDFKVlJ6UjJjR1YzZEZoV1YzbG1lamRZY0hBd1ExazJZamRyUVRSS2N6TklhR3d3YkZJMFdUQTFMemN2UVVkQ2FEZFZNSGczUkhaTVozUXpVM00wYm5GTFZTdFhXRXBTVHpKWVFVRnZSME4xZFRWR1RGcHJRVWhYY1RSUVFtMXphSFY2Y1ZsdmNucHhlbGhGWVZWVlpFUlVkVXhDTW1nNWFIZ3dXRWhQUmxwUk16bHVkbTlPUjJaT2R5OTRTVmRaZEhSUGRYZHZhMncyTVZsb1JVeFZlRmQxU1ZSRmMwTlVhM2xtTVRNd09IazVSbFJzWlRKeVYyZEVlSEZNYTBSUFNXVXlPRWwzUzJSQkwySXdWVUl5VEZGbVRWcHdWemwyUTNCSkwybHlWek5uYmpaeU5WWjNWMjB2U1dweWJtNDNSelJrVmpadVYzcFRkMGhQUTJSdWEwMTRNRXQ1VVVOa0wxQjFaWEpUYjNSdVEwOXRTMDEzWlRSTGJqaERkMU5YVVRRNGRURkRNbTFpV1VzeGRYTlpOM1YzUFQwaUxDSndkV0pzYVdOTFpYa2lPaUl0TFMwdExVSkZSMGxPSUZCVlFreEpReUJMUlZrdExTMHRMVnh1VFVsSlFrbHFRVTVDWjJ0eGFHdHBSemwzTUVKQlVVVkdRVUZQUTBGUk9FRk5TVWxDUTJkTFEwRlJSVUZ6TkhKdlVIcDFhV1JNZVhOMmIxWTJkemxhTkZ4dVdHRmliME5tWTJNeGFHZFZhQ3N3V1VkS2NFNURSVXhyTjBaTFF5OTJhemR6ZERsR05tY3dUMjlrU0VSbGVYZFJXa2hLZFU1TVpsUnNRbEJHUTJOaU5seHVObTlzVEZOeWNGQTRjbFUzU0d4SGJsRkVSMFJNYVhkS1EyaGtSRGRVVUdSM2FXdHBkMHRGY201aldqaEdaalZsU25vd2RETmlUWFpyVDJaVVluSkJiRnh1WWtGQ1kwbzVNVmxVT1hKdVVXOXFkVWN4UldKUVRqaEZWblI2TWxZNE5IZHViR2Q0TUhCd2JEVjRPSFpOYlhwcE1ISnVibEZVV1VGamJ6WnFhMnBJTTF4dVRuTlVkWE4xUzFkdlJGUjVNWE5yZGtSUk9IbEJZV0ptWTNNME4zWnNRazAwU0RGT1JFNHZSSFJhWWxZdllubDJia0o2YkM4eFZrVnpURmRqWlZWcFRGeHVSWEYxT0VkeWF5dFFVRGQyUkdSd2JFUjNjWFpQV2t4RmRYazNkamhuUm01U09WUlVSV3ByTlVvNWRuWlVTR2RtU25VemVubEVPR2xLWTBSRE5YcHFPVnh1YjFGSlJFRlJRVUpjYmkwdExTMHRSVTVFSUZCVlFreEpReUJMUlZrdExTMHRMVnh1SWl3aWEyVjVVMmxuYm1GMGRYSmxJam9pWlhsS2VtRlhaSFZaV0ZJeFkyMVZhVTlwU2pCUldIQjJXVE5LVms1NmFGaFNSMlJzVVRKb2NtTklXa1ZVVlRsRldqQktXVTFGUmtaVFJFNUZVMGhLYkUxclRUTkxNSEJFVkROR2VGTnROVVJVVlRWVlltMDFiVnBGUm5sWldIQjZaRVJqTVZaSGFFeFBXRUpVVWtacmRrd3diek5aTUZaSlVteFdWRXd5T1VoV1JXeHNWa1ZPTUZSSE1WWlJNR04zVkd4R2JGa3pTblJUUm1zMFZVWk9hMVpWU2pCVU1WbDNZbXQwY0ZSclZuQmpia0poVFZjNWFtSldiSEZaYTNob1UyeHNWV0pGUmtWWGJVWnZWakZLVUZkcWJGSmhXRVp1V2xkb1EyRnVRak5TUjNNd1lWWkpOVTVXVmxkV1ZUVnlUMGhLYjFsVlRYbGhiVGcwVjBkYWVGbHFWbFppYlhoeFpFWkZkMDU1Y3pCaFZsSkpWRVpPTm1WRk1IcGxWWFJ2VFVaR1ZtRXdWVFJSVnpsSFVsaEtVRTFZUmxCU01WcFJVMVJDTmxsV2FIcFdWWEJ0WTBSU2JFMVVRazlPVjNSU1ZucFdUMU5XWTNaU1ZYUkZVMGhzYlU5VmJGaGtNMUl3WTFWc1lXTlhSakJTYTA1RVlVWmtjbUo2VmtSU00wSllUREkxUmsxWVl6SmxWM1JKVlZoQk1sVXhTbEppU0Zwd1VrVXdNRlpFVWt0VU1rWnNVVmQwYzFSV1VrMVVWV055V1RCYVRHSXpaRTlUVm05NVlraE9SR1JzVG5aUmFrWmFaVmRPVGxOVlNteGFiRXB1Wld0U2RVMHhSVGxRVTBselNXMWtjMkl5U21oaVJYUnNaVlZzYTBscWIybFpiVkpzV2xSVk1rNVVXWGRaTWxwcFRrUk9hazlYU1hsUFIwcHRUMVJvYkZsWFRtaGFiVVV5VGtSWmFXWlJQVDBpZlE9PSJ9
`, mockServer.URL)

	testMultiChannelLicense := fmt.Sprintf(`apiVersion: kots.io/v1beta1
kind: License
metadata:
  name: testcustomer
spec:
  appSlug: my-app
  channelID: 1vusIYZLAVxMG6q760OJmRKj5i5
  channelName: My Channel
  customerName: Test Customer
  endpoint: %s
  channels:
  - channelId: 1vusIYZLAVxMG6q760OJmRKj5i5
    channelName: My Channel
    channelSlug: my-channel
    isDefault: true
    isSemverRequired: false
  entitlements:
    expires_at:
      description: License Expiration
      title: Expiration
      value: "2030-07-27T00:00:00Z"
      valueType: String
  isAirgapSupported: true
  isGitOpsSupported: true
  isSnapshotSupported: true
  licenseID: 1vusOokxAVp1tkRGuyxnF23PJcq
  licenseSequence: 7
  licenseType: prod
  signature: eyJsaWNlbnNlRGF0YSI6ImV5SmhjR2xXWlhKemFXOXVJam9pYTI5MGN5NXBieTkyTVdKbGRHRXhJaXdpYTJsdVpDSTZJa3hwWTJWdWMyVWlMQ0p0WlhSaFpHRjBZU0k2ZXlKdVlXMWxJam9pZEdWemRHTjFjM1J2YldWeUluMHNJbk53WldNaU9uc2liR2xqWlc1elpVbEVJam9pTVhaMWMwOXZhM2hCVm5BeGRHdFNSM1Y1ZUc1R01qTlFTbU54SWl3aWJHbGpaVzV6WlZSNWNHVWlPaUp3Y205a0lpd2lZM1Z6ZEc5dFpYSk9ZVzFsSWpvaVZHVnpkQ0JEZFhOMGIyMWxjaUlzSW1Gd2NGTnNkV2NpT2lKdGVTMWhjSEFpTENKamFHRnVibVZzU1VRaU9pSXhkblZ6U1ZsYVRFRldlRTFITm5FM05qQlBTbTFTUzJvMWFUVWlMQ0pqYUdGdWJtVnNUbUZ0WlNJNklrMTVJRU5vWVc1dVpXd2lMQ0pzYVdObGJuTmxVMlZ4ZFdWdVkyVWlPamNzSW1WdVpIQnZhVzUwSWpvaWFIUjBjSE02THk5eVpYQnNhV05oZEdWa0xtRndjQ0lzSW1WdWRHbDBiR1Z0Wlc1MGN5STZleUppYjI5c1gyWnBaV3hrSWpwN0luUnBkR3hsSWpvaVFtOXZiQ0JHYVdWc1pDSXNJblpoYkhWbElqcDBjblZsTENKMllXeDFaVlI1Y0dVaU9pSkNiMjlzWldGdUluMHNJbVY0Y0dseVpYTmZZWFFpT25zaWRHbDBiR1VpT2lKRmVIQnBjbUYwYVc5dUlpd2laR1Z6WTNKcGNIUnBiMjRpT2lKTWFXTmxibk5sSUVWNGNHbHlZWFJwYjI0aUxDSjJZV3gxWlNJNklqSXdNekF0TURjdE1qZFVNREE2TURBNk1EQmFJaXdpZG1Gc2RXVlVlWEJsSWpvaVUzUnlhVzVuSW4wc0ltaHBaR1JsYmw5bWFXVnNaQ0k2ZXlKMGFYUnNaU0k2SWtocFpHUmxiaUJHYVdWc1pDSXNJblpoYkhWbElqb2lkR2hwY3lCcGN5QnpaV055WlhRaUxDSjJZV3gxWlZSNWNHVWlPaUpUZEhKcGJtY2lMQ0pwYzBocFpHUmxiaUk2ZEhKMVpYMHNJbWx1ZEY5bWFXVnNaQ0k2ZXlKMGFYUnNaU0k2SWtsdWRDQkdhV1ZzWkNJc0luWmhiSFZsSWpveE1qTXNJblpoYkhWbFZIbHdaU0k2SWtsdWRHVm5aWElpZlN3aWMzUnlhVzVuWDJacFpXeGtJanA3SW5ScGRHeGxJam9pVTNSeWFXNW5SbWxsYkdRaUxDSjJZV3gxWlNJNkluTnBibWRzWlNCc2FXNWxJSFJsZUhRaUxDSjJZV3gxWlZSNWNHVWlPaUpUZEhKcGJtY2lmU3dpZEdWNGRGOW1hV1ZzWkNJNmV5SjBhWFJzWlNJNklsUmxlSFFnUm1sbGJHUWlMQ0oyWVd4MVpTSTZJbTExYkhScFhHNXNhVzVsWEc1MFpYaDBJaXdpZG1Gc2RXVlVlWEJsSWpvaVZHVjRkQ0o5ZlN3aWFYTkJhWEpuWVhCVGRYQndiM0owWldRaU9uUnlkV1VzSW1selIybDBUM0J6VTNWd2NHOXlkR1ZrSWpwMGNuVmxMQ0pwYzFOdVlYQnphRzkwVTNWd2NHOXlkR1ZrSWpwMGNuVmxmWDA9IiwiaW5uZXJTaWduYXR1cmUiOiJleUpzYVdObGJuTmxVMmxuYm1GMGRYSmxJam9pYUhneE1XTXZUR1ozUTNoVE5YRmtRWEJGU1hGdVRrMU9NMHBLYTJzNFZHZFhSVVpzVDFKVlJ6UjJjR1YzZEZoV1YzbG1lamRZY0hBd1ExazJZamRyUVRSS2N6TklhR3d3YkZJMFdUQTFMemN2UVVkQ2FEZFZNSGczUkhaTVozUXpVM00wYm5GTFZTdFhXRXBTVHpKWVFVRnZSME4xZFRWR1RGcHJRVWhYY1RSUVFtMXphSFY2Y1ZsdmNucHhlbGhGWVZWVlpFUlVkVXhDTW1nNWFIZ3dXRWhQUmxwUk16bHVkbTlPUjJaT2R5OTRTVmRaZEhSUGRYZHZhMncyTVZsb1JVeFZlRmQxU1ZSRmMwTlVhM2xtTVRNd09IazVSbFJzWlRKeVYyZEVlSEZNYTBSUFNXVXlPRWwzUzJSQkwySXdWVUl5VEZGbVRWcHdWemwyUTNCSkwybHlWek5uYmpaeU5WWjNWMjB2U1dweWJtNDNSelJrVmpadVYzcFRkMGhQUTJSdWEwMTRNRXQ1VVVOa0wxQjFaWEpUYjNSdVEwOXRTMDEzWlRSTGJqaERkMU5YVVRRNGRURkRNbTFpV1VzeGRYTlpOM1YzUFQwaUxDSndkV0pzYVdOTFpYa2lPaUl0TFMwdExVSkZSMGxPSUZCVlFreEpReUJMUlZrdExTMHRMVnh1VFVsSlFrbHFRVTVDWjJ0eGFHdHBSemwzTUVKQlVVVkdRVUZQUTBGUk9FRk5TVWxDUTJkTFEwRlJSVUZ6TkhKdlVIcDFhV1JNZVhOMmIxWTJkemxhTkZ4dVdHRmliME5tWTJNeGFHZFZhQ3N3V1VkS2NFNURSVXhyTjBaTFF5OTJhemR6ZERsR05tY3dUMjlrU0VSbGVYZFJXa2hLZFU1TVpsUnNRbEJHUTJOaU5seHVObTlzVEZOeWNGQTRjbFUzU0d4SGJsRkVSMFJNYVhkS1EyaGtSRGRVVUdSM2FXdHBkMHRGY201aldqaEdaalZsU25vd2RETmlUWFpyVDJaVVluSkJiRnh1WWtGQ1kwbzVNVmxVT1hKdVVXOXFkVWN4UldKUVRqaEZWblI2TWxZNE5IZHViR2Q0TUhCd2JEVjRPSFpOYlhwcE1ISnVibEZVV1VGamJ6WnFhMnBJTTF4dVRuTlVkWE4xUzFkdlJGUjVNWE5yZGtSUk9IbEJZV0ptWTNNME4zWnNRazAwU0RGT1JFNHZSSFJhWWxZdllubDJia0o2YkM4eFZrVnpURmRqWlZWcFRGeHVSWEYxT0VkeWF5dFFVRGQyUkdSd2JFUjNjWFpQV2t4RmRYazNkamhuUm01U09WUlVSV3ByTlVvNWRuWlVTR2RtU25VemVubEVPR2xLWTBSRE5YcHFPVnh1YjFGSlJFRlJRVUpjYmkwdExTMHRSVTVFSUZCVlFreEpReUJMUlZrdExTMHRMVnh1SWl3aWEyVjVVMmxuYm1GMGRYSmxJam9pWlhsS2VtRlhaSFZaV0ZJeFkyMVZhVTlwU2pCUldIQjJXVE5LVms1NmFGaFNSMlJzVVRKb2NtTklXa1ZVVlRsRldqQktXVTFGUmtaVFJFNUZVMGhLYkUxclRUTkxNSEJFVkROR2VGTnROVVJVVlRWVlltMDFiVnBGUm5sWldIQjZaRVJqTVZaSGFFeFBXRUpVVWtacmRrd3diek5aTUZaSlVteFdWRXd5T1VoV1JXeHNWa1ZPTUZSSE1WWlJNR04zVkd4R2JGa3pTblJUUm1zMFZVWk9hMVpWU2pCVU1WbDNZbXQwY0ZSclZuQmpia0poVFZjNWFtSldiSEZaYTNob1UyeHNWV0pGUmtWWGJVWnZWakZLVUZkcWJGSmhXRVp1V2xkb1EyRnVRak5TUjNNd1lWWkpOVTVXVmxkV1ZUVnlUMGhLYjFsVlRYbGhiVGcwVjBkYWVGbHFWbFppYlhoeFpFWkZkMDU1Y3pCaFZsSkpWRVpPTm1WRk1IcGxWWFJ2VFVaR1ZtRXdWVFJSVnpsSFVsaEtVRTFZUmxCU01WcFJVMVJDTmxsV2FIcFdWWEJ0WTBSU2JFMVVRazlPVjNSU1ZucFdUMU5XWTNaU1ZYUkZVMGhzYlU5VmJGaGtNMUl3WTFWc1lXTlhSakJTYTA1RVlVWmtjbUo2VmtSU00wSllUREkxUmsxWVl6SmxWM1JKVlZoQk1sVXhTbEppU0Zwd1VrVXdNRlpFVWt0VU1rWnNVVmQwYzFSV1VrMVVWV055V1RCYVRHSXpaRTlUVm05NVlraE9SR1JzVG5aUmFrWmFaVmRPVGxOVlNteGFiRXB1Wld0U2RVMHhSVGxRVTBselNXMWtjMkl5U21oaVJYUnNaVlZzYTBscWIybFpiVkpzV2xSVk1rNVVXWGRaTWxwcFRrUk9hazlYU1hsUFIwcHRUMVJvYkZsWFRtaGFiVVV5VGtSWmFXWlJQVDBpZlE9PSJ9
`, mockServer.URL)

	onlineApp := &apptypes.App{
		ID:       "app-id",
		Slug:     "app-slug",
		Name:     "app-name",
		IsAirgap: false,
		IsGitOps: false,
		License:  testLicense,
	}

	airgapApp := &apptypes.App{
		ID:       "app-id",
		Slug:     "app-slug",
		Name:     "app-name",
		IsAirgap: true,
		IsGitOps: false,
		License:  testLicense,
	}

	airgapAppMultiChannel := &apptypes.App{
		ID:       "app-id",
		Slug:     "app-slug",
		Name:     "app-name",
		IsAirgap: true,
		IsGitOps: false,
		License:  testMultiChannelLicense,
	}

	type args struct {
		app     *apptypes.App
		request handlers.StartUpgradeServiceRequest
	}
	tests := []struct {
		name                  string
		args                  args
		mockStoreExpectations func()
		wantParams            *upgradeservicetypes.UpgradeServiceParams
	}{
		{
			name: "online",
			args: args{
				app: onlineApp,
				request: handlers.StartUpgradeServiceRequest{
					VersionLabel: "1.0.0",
					UpdateCursor: "1",
					ChannelID:    "channel-id",
				},
			},
			mockStoreExpectations: func() {
				mockStore.EXPECT().GetRegistryDetailsForApp(onlineApp.ID).Return(registrytypes.RegistrySettings{}, nil)
				mockStore.EXPECT().GetAppVersionBaseArchive(onlineApp.ID, "1.0.0").Return("base-archive", int64(1), nil)
				mockStore.EXPECT().GetNextAppSequence(onlineApp.ID).Return(int64(2), nil)
			},
			wantParams: &upgradeservicetypes.UpgradeServiceParams{
				Port: "", // port is random, we just check it's not empty

				AppID:       onlineApp.ID,
				AppSlug:     onlineApp.Slug,
				AppName:     onlineApp.Name,
				AppIsAirgap: onlineApp.IsAirgap,
				AppIsGitOps: onlineApp.IsGitOps,
				AppLicense:  onlineApp.License,
				AppArchive:  "base-archive",

				Source:       "Upstream Update",
				BaseSequence: 1,
				NextSequence: 2,

				UpdateVersionLabel: "1.0.0",
				UpdateCursor:       "1",
				UpdateChannelID:    "channel-id",
				UpdateChannelSlug:  "channel-slug",
				UpdateECVersion:    "online-update-ec-version",
				UpdateKOTSBin:      "", // tmp file name is random, we just check it's not empty
				UpdateAirgapBundle: "",

				CurrentECVersion: "current-ec-version",

				RegistryEndpoint:   "",
				RegistryUsername:   "",
				RegistryPassword:   "",
				RegistryNamespace:  "",
				RegistryIsReadOnly: false,

				ReportingInfo: reporting.GetReportingInfo(onlineApp.ID),
			},
		},
		{
			name: "airgap",
			args: args{
				app: airgapApp,
				request: handlers.StartUpgradeServiceRequest{
					VersionLabel: "1.0.0",
					UpdateCursor: "1",
					ChannelID:    "channel-id",
				},
			},
			mockStoreExpectations: func() {
				mockStore.EXPECT().GetRegistryDetailsForApp(airgapApp.ID).Return(registrytypes.RegistrySettings{
					Hostname:   "hostname",
					Username:   "username",
					Password:   "password",
					Namespace:  "namespace",
					IsReadOnly: false,
				}, nil)
				mockStore.EXPECT().GetAppVersionBaseArchive(airgapApp.ID, "1.0.0").Return("base-archive", int64(1), nil)
				mockStore.EXPECT().GetNextAppSequence(airgapApp.ID).Return(int64(2), nil)
			},
			wantParams: &upgradeservicetypes.UpgradeServiceParams{
				Port: "", // port is random, we just check it's not empty

				AppID:       airgapApp.ID,
				AppSlug:     airgapApp.Slug,
				AppName:     airgapApp.Name,
				AppIsAirgap: airgapApp.IsAirgap,
				AppIsGitOps: airgapApp.IsGitOps,
				AppLicense:  airgapApp.License,
				AppArchive:  "base-archive",

				Source:       "Airgap Update",
				BaseSequence: 1,
				NextSequence: 2,

				UpdateVersionLabel: "1.0.0",
				UpdateCursor:       "1",
				UpdateChannelID:    "channel-id",
				UpdateChannelSlug:  "channel-slug",
				UpdateECVersion:    "airgap-update-ec-version",
				UpdateKOTSBin:      "", // tmp file name is random, we just check it's not empty
				UpdateAirgapBundle: updateAirgapBundle,

				CurrentECVersion: "current-ec-version",

				RegistryEndpoint:   "hostname",
				RegistryUsername:   "username",
				RegistryPassword:   "password",
				RegistryNamespace:  "namespace",
				RegistryIsReadOnly: false,

				ReportingInfo: reporting.GetReportingInfo(airgapApp.ID),
			},
		},
		{
			name: "airgap with multi-channel license",
			args: args{
				app: airgapAppMultiChannel,
				request: handlers.StartUpgradeServiceRequest{
					VersionLabel: "1.0.0",
					UpdateCursor: "1",
					ChannelID:    "channel-id",
				},
			},
			mockStoreExpectations: func() {
				mockStore.EXPECT().GetRegistryDetailsForApp(airgapAppMultiChannel.ID).Return(registrytypes.RegistrySettings{
					Hostname:   "hostname",
					Username:   "username",
					Password:   "password",
					Namespace:  "namespace",
					IsReadOnly: false,
				}, nil)
				mockStore.EXPECT().GetAppVersionBaseArchive(airgapAppMultiChannel.ID, "1.0.0").Return("base-archive", int64(1), nil)
				mockStore.EXPECT().GetNextAppSequence(airgapAppMultiChannel.ID).Return(int64(2), nil)
			},
			wantParams: &upgradeservicetypes.UpgradeServiceParams{
				Port: "", // port is random, we just check it's not empty

				AppID:       airgapAppMultiChannel.ID,
				AppSlug:     airgapAppMultiChannel.Slug,
				AppName:     airgapAppMultiChannel.Name,
				AppIsAirgap: airgapAppMultiChannel.IsAirgap,
				AppIsGitOps: airgapAppMultiChannel.IsGitOps,
				AppLicense:  airgapAppMultiChannel.License,
				AppArchive:  "base-archive",

				Source:       "Airgap Update",
				BaseSequence: 1,
				NextSequence: 2,

				UpdateVersionLabel: "1.0.0",
				UpdateCursor:       "1",
				UpdateChannelID:    "channel-id",
				UpdateChannelSlug:  "channel-slug",
				UpdateECVersion:    "airgap-update-ec-version",
				UpdateKOTSBin:      "", // tmp file name is random, we just check it's not empty
				UpdateAirgapBundle: updateAirgapBundle,

				CurrentECVersion: "current-ec-version",

				RegistryEndpoint:   "hostname",
				RegistryUsername:   "username",
				RegistryPassword:   "password",
				RegistryNamespace:  "namespace",
				RegistryIsReadOnly: false,

				ReportingInfo: reporting.GetReportingInfo(airgapApp.ID),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockStoreExpectations()

			gotParams, err := handlers.GetUpgradeServiceParams(mockStore, tt.args.app, tt.args.request)
			require.NoError(t, err)

			assert.NotEqual(t, "", gotParams.Port)
			assert.NotEqual(t, "", gotParams.UpdateKOTSBin)

			tt.wantParams.Port = gotParams.Port
			tt.wantParams.UpdateKOTSBin = gotParams.UpdateKOTSBin
			assert.Equal(t, tt.wantParams, gotParams)

			err = upgradeservice.Start(*gotParams)
			require.NoError(t, err)

			// test proxying to the ping endpoint
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", fmt.Sprintf("http://kotsadm:3000/api/v1/upgrade-service/app/%s/ping", gotParams.AppSlug), nil)
			r = mux.SetURLVars(r, map[string]string{"appSlug": gotParams.AppSlug})
			upgradeservice.Proxy(w, r)
			assert.Equal(t, http.StatusOK, w.Code)

			// test GET proxying to an endpoint that is unknown to the current kots version
			w = httptest.NewRecorder()
			r = httptest.NewRequest("GET", fmt.Sprintf("http://kotsadm:3000/api/v1/upgrade-service/app/%s/unknown", gotParams.AppSlug), nil)
			r = mux.SetURLVars(r, map[string]string{"appSlug": gotParams.AppSlug})
			upgradeservice.Proxy(w, r)
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "unknown GET body", w.Body.String())

			// test POST proxying to an endpoint that is unknown to the current kots version
			w = httptest.NewRecorder()
			r = httptest.NewRequest("POST", fmt.Sprintf("http://kotsadm:3000/api/v1/upgrade-service/app/%s/unknown", gotParams.AppSlug), nil)
			r = mux.SetURLVars(r, map[string]string{"appSlug": gotParams.AppSlug})
			upgradeservice.Proxy(w, r)
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "unknown POST body", w.Body.String())

			// test proxying to a non-existing endpoint
			w = httptest.NewRecorder()
			r = httptest.NewRequest("GET", fmt.Sprintf("http://kotsadm:3000/api/v1/upgrade-service/app/%s/non-existing", gotParams.AppSlug), nil)
			r = mux.SetURLVars(r, map[string]string{"appSlug": gotParams.AppSlug})
			upgradeservice.Proxy(w, r)
			assert.Equal(t, http.StatusNotFound, w.Code)

			upgradeservice.Stop(gotParams.AppSlug)
		})
	}
}

func mockUpdateAirgapBundle(t *testing.T) string {
	bundle := filepath.Join(t.TempDir(), "update-bundle.airgap")
	defer os.Remove(bundle)

	f, err := os.Create(bundle)
	require.NoError(t, err)
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	airgapYAML := `apiVersion: kots.io/v1beta1
kind: Airgap
spec:
  appSlug: app-slug
  channelID: channel-id
  updateCursor: "1"
  embeddedClusterArtifacts:
    additionalArtifacts:
      kots: embedded-cluster/artifacts/kots.tar.gz
    metadata: embedded-cluster/version-metadata.json`

	kotsTGZ := mockKOTSBinary(t)

	metadataJSON := `{
  "Versions": {
    "Installer": "airgap-update-ec-version"
  }
}`

	err = tw.WriteHeader(&tar.Header{
		Name: "airgap.yaml",
		Mode: 0644,
		Size: int64(len(airgapYAML)),
	})
	require.NoError(t, err)

	_, err = tw.Write([]byte(airgapYAML))
	require.NoError(t, err)

	err = tw.WriteHeader(&tar.Header{
		Name: "embedded-cluster/artifacts/kots.tar.gz",
		Mode: 0755,
		Size: int64(len(kotsTGZ)),
	})
	require.NoError(t, err)

	_, err = tw.Write(kotsTGZ)
	require.NoError(t, err)

	err = tw.WriteHeader(&tar.Header{
		Name: "embedded-cluster/version-metadata.json",
		Mode: 0644,
		Size: int64(len(metadataJSON)),
	})
	require.NoError(t, err)

	_, err = tw.Write([]byte(metadataJSON))
	require.NoError(t, err)

	tw.Close()
	gw.Close()

	err = update.InitAvailableUpdatesDir()
	require.NoError(t, err)

	err = update.RegisterAirgapUpdate("app-slug", bundle)
	require.NoError(t, err)

	airgapUpdate, err := update.GetAirgapUpdate("app-slug", "channel-id", "1")
	require.NoError(t, err)

	return airgapUpdate
}

// use the test executable to mock the kots binary
// reference: https://abhinavg.net/2022/05/15/hijack-testmain
func mockKOTSBinary(t *testing.T) []byte {
	testExe, err := os.Executable()
	require.NoError(t, err)

	kotsBin, err := os.ReadFile(testExe)
	require.NoError(t, err)

	buf := bytes.NewBuffer(nil)
	gw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gw)

	err = tw.WriteHeader(&tar.Header{
		Name: "kots",
		Mode: 0755,
		Size: int64(len(kotsBin)),
	})
	require.NoError(t, err)

	_, err = tw.Write(kotsBin)
	require.NoError(t, err)

	tw.Close()
	gw.Close()

	return buf.Bytes()
}

func mockUpgradeServiceCmd() {
	wantArgs := []string{"upgrade-service", "start", "-"}
	if gotArgs := os.Args[1:]; !slices.Equal(wantArgs, gotArgs) {
		log.Fatalf(`expected arguments %q, got %q`, wantArgs, gotArgs)
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Failed to read stdin: %v", err)
	}

	var params struct {
		Port    string `yaml:"port"`
		AppSlug string `yaml:"appSlug"`
	}
	if err := yaml.Unmarshal(data, &params); err != nil {
		log.Fatalf("Failed to unmarshal params YAML: %v", err)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fmt.Sprintf("/api/v1/upgrade-service/app/%s/ping", params.AppSlug) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}
		if r.URL.Path == fmt.Sprintf("/api/v1/upgrade-service/app/%s/unknown", params.AppSlug) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("unknown %s body", r.Method)))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	http.HandleFunc("/", handler)
	fmt.Println("starting mock upgrade service on port", params.Port)

	if err := http.ListenAndServe(":"+params.Port, nil); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
