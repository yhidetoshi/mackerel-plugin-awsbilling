package mpawsbilling

import "testing"

func TestGraphDefinition(t *testing.T) {
	var plugin AwsBillingPlugin
	graphdef := plugin.GraphDefinition()

	expected := 1
	if actual := len(graphdef); actual != expected {
		t.Errorf("GraphDefinition(): %d should be %d", actual, expected)
	}
}
