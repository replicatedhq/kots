package testim

import "testing"

func TestTestimURLRegexp(t *testing.T) {
	out := `    Starting Testim.io CLI
    - Starting testim ngrok tunnel...
    - Installing ngrok before first usage...
    ✔ Installing ngrok before first usage...
    ✔ Starting testim ngrok tunnel...
    Run anonymous (Label: type=existing cluster, env=online, phase=new install, rbac=minimal rbac) test plan with default configs, Project: wpYAooUimFDgQxY73r17, Branch: master (wmvPajNTQzLdU4AO)
    -----------------------------------------------------------------------------------
    Test list:
         1 : type=existing cluster, env=online, phase=new install, rbac=minimal rbac (3KsnsQNZ2U4cpbxQ) 
    -----------------------------------------------------------------------------------
     Test "type=existing cluster, env=online, phase=new install, rbac=minimal rbac" started (3KsnsQNZ2U4cpbxQ) url: https://app.testim.io/#/project/wpYAooUimFDgQxY73r17/branch/master/test/3KsnsQNZ2U4cpbxQ?result-id=URSSXDTZ7rv4EscQ
     Get chrome slot from Testim-grid
     Get browser to run type=existing cluster, env=online, phase=new install, rbac=minimal rbac
     Wait for test start
     Wait for test complete`
	want := "https://app.testim.io/#/project/wpYAooUimFDgQxY73r17/branch/master/test/3KsnsQNZ2U4cpbxQ?result-id=URSSXDTZ7rv4EscQ"
	got := TestimURLRegexp.FindString(out)
	if got != want {
		t.Errorf("TestimURLRegexp.FindString() = %v, want %v", got, want)
	}
}
