package listener

import "testing"

func TestNew(t *testing.T) {

}

func TestListen(t *testing.T) {

}

func TestHandleEvents(t *testing.T) {

}

func TestTreatEvent(t *testing.T) {

}

func TestDelegate(t *testing.T) {

}

func TestGetServiceInfo(t *testing.T) {

}

func TestGetServiceInfoFromCache(t *testing.T) {

}

func TestGetServiceInfoFromInspect(t *testing.T) {

}

func TestIsDNSEvent(t *testing.T) {

}

func TestGracefulStop(t *testing.T) {

}

func TestHandleErrors(t *testing.T) {

}

func TestStop(t *testing.T) {

}

func TestBackoffWait(t *testing.T) {

}

func TestSandmanServiceCheck(t *testing.T) {
	ss := &SandmanService{HostName: "t.test.com", ServiceName: "testservice", Tags: []string{"internal", "intranet"}}

	contextTags := []string{"internet"}
	result, err := ss.check(contextTags)

	if result {
		t.Errorf("Expecting the error 'No matching tags found'. Got success for ss := '%v' and context tags := '%v'", ss, contextTags)
	}

	if len(err) > 0 && err[0] != "No matching tags found" {
		t.Errorf("Expecting the error 'No matching tags found'. Got '%v' for ss := '%v' and context tags := '%v'", err[0], ss, contextTags)
	}

	if !result && len(err) == 0 {
		t.Errorf("Got non-success execution, but no errors reported for ss := '%v' and context tags := '%v'", ss, contextTags)
	}

	contextTags = []string{"intranet"}
	result, err = ss.check(contextTags)

	if !result && len(err) == 0 {
		t.Errorf("Got non-success execution, but no errors reported for ss := '%v' and context tags := '%v'", ss, contextTags)
	}

	if !result && len(err) > 0 {
		t.Errorf("Expecting success. Got non-success execution for ss := '%v' and context tags := '%v' with errors: %v", ss, contextTags, err)
	}
}
