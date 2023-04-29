package facts

type CustomFactsGatherer func() (any, error)

var customFactsGatherers = map[string]CustomFactsGatherer{}

func RegisterCustomFacts(name string, fn CustomFactsGatherer) {
	customFactsGatherers[name] = fn
}
