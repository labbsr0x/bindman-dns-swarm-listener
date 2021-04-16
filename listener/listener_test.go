package listener

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	//ss := &SandmanService{HostName: "t.test.com", ServiceName: "testservice", Tags: []string{"internal", "intranet"}}
	//
	//contextTags := []string{"internet"}
	//result, err := ss.check(contextTags)
	//
	//if result {
	//	t.Errorf("Expecting the error 'No matching tags found'. Got success for ss := '%v' and context tags := '%v'", ss, contextTags)
	//}
	//
	//if len(err) > 0 && err[0] != "No matching tags found" {
	//	t.Errorf("Expecting the error 'No matching tags found'. Got '%v' for ss := '%v' and context tags := '%v'", err[0], ss, contextTags)
	//}
	//
	//if !result && len(err) == 0 {
	//	t.Errorf("Got non-success execution, but no errors reported for ss := '%v' and context tags := '%v'", ss, contextTags)
	//}
	//
	//contextTags = []string{"intranet"}
	//result, err = ss.check(contextTags)
	//
	//if !result && len(err) == 0 {
	//	t.Errorf("Got non-success execution, but no errors reported for ss := '%v' and context tags := '%v'", ss, contextTags)
	//}
	//
	//if !result && len(err) > 0 {
	//	t.Errorf("Expecting success. Got non-success execution for ss := '%v' and context tags := '%v' with errors: %v", ss, contextTags, err)
	//}
}

func TestExtractLabelNamesForHostname(t *testing.T) {
	labelsMap := make(map[string]string)
	labelsMap["traefik.frontend.rule"] = "v1.x"
	labelsMap["traefik.frontenda.rule"] = "v1.x"
	labelsMap[" traefik.frontend.rule"] = "v1.x"
	labelsMap["traefika..frontend.rulse "] = "v1.x"
	labelsMap["traefik.http.routers.whoami.rule"] = "v2.x"
	labelsMap["traefik.http.routers.asda00.rule"] = "v2.x"
	labelsMap["traefik.http.routers.asda--asdad.rule"] = "v2.x"
	labelsMap["traefik.http.routers.asda_Adsda.rule"] = "v2.x"
	labelsMap["traefiks.https.routers.whoami.rules"] = "v2.x"
	labelsMap[" traefiks.https.routers.whoami.rules"] = "v2.x"
	labelsMap[" traefiks.https.routers.whoami.rules "] = "v2.x"

	labelsExpectedResult := []string{"traefik.frontend.rule", "traefik.http.routers.whoami.rule", "traefik.http.routers.asda_Adsda.rule", "traefik.http.routers.asda00.rule", "traefik.http.routers.asda--asdad.rule"}
	labels := findTraefikLabelForHostNames(labelsMap)

	assert.ElementsMatch(t, labelsExpectedResult, labels)
}

func TestTextHostNameExtraction(t *testing.T) {
	values := []string{
		"80",
		"traefik-net",
		"internal_http,internal_https",
		"true",
		"Host(`anonovo2021.labbs.com.br`, `anonovo20212.labbs.com.br`)",
		"Host:anonove2023.labbs.com.br",
		"Host:anonovo1900.labbs.com.br; Host:anonovo1901.labbs.com.br ;Host:anonovo1902.labbs.com.br",
		"Host(`anonovo2020.labbs.com.br`)",
		"Host(`example.com`) || (Host(`example.org`) && Path(`/traefik`))",
	}

	expectedValues := make([][]string, 9)
	expectedValues[0] = nil
	expectedValues[1] = nil
	expectedValues[2] = nil
	expectedValues[3] = nil
	expectedValues[4] = []string{"anonovo2021.labbs.com.br", "anonovo20212.labbs.com.br"}
	expectedValues[5] = []string{"anonove2023.labbs.com.br"}
	expectedValues[6] = []string{"anonovo1900.labbs.com.br", "anonovo1901.labbs.com.br", "anonovo1902.labbs.com.br"}
	expectedValues[7] = []string{"anonovo2020.labbs.com.br"}
	expectedValues[8] = []string{"example.com", "example.org"}

	for i, text := range values {
		results := getHostNamesFromLabelRegex(text)
		assert.ElementsMatch(t, expectedValues[i], results)
	}
}
