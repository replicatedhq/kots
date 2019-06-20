package specs

import (
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/templates"
)

func NewIDPatcher(logger log.Logger) *IDPatcher {
	return &IDPatcher{
		Logger: log.With(logger, "struct", "idpatcher"),
	}
}

type IDPatcher struct {
	Logger log.Logger
}

type idSet map[string]interface{}

func (s idSet) contains(id string) bool {
	_, ok := s[id]
	return ok

}

func (s idSet) add(id string) {
	s[id] = true
}

func (p *IDPatcher) EnsureAllStepsHaveUniqueIDs(lc api.Lifecycle) api.Lifecycle {

	newLc := api.Lifecycle{V1: []api.Step{}}
	seenIds := make(idSet)
	for _, step := range lc.V1 {
		id := step.Shared().ID
		if id != "" && !seenIds.contains(id) {
			// the id is unique, add it to the ones we've seen, and append this step to the new lifecycle
			seenIds.add(id)
			newLc.V1 = append(newLc.V1, step)
			continue
		}

		// the current step's id is missing or a duplicate, so generate a new one
		newID := p.generateID(seenIds, step)
		level.Debug(p.Logger).Log("event", "id.generate", "id", newID)
		seenIds.add(newID)
		step.Shared().ID = newID
		newLc.V1 = append(newLc.V1, step)
	}
	return newLc
}

func (p *IDPatcher) generateID(seenIds idSet, step api.Step) string {
	// try with the $shortname
	candidateID := fmt.Sprintf("%s", step.ShortName())
	if !seenIds.contains(candidateID) {
		return candidateID
	}

	// try ${shortname}-2 ${shortname}-3 up to 99
	i := 2
	for i < 100 {
		candidateID := fmt.Sprintf("%s-%d", step.ShortName(), i)
		if !seenIds.contains(candidateID) {
			return candidateID
		}
		i++
	}

	// hack, just get a random one
	return fmt.Sprintf(
		"%s-%s",
		step.ShortName(),
		(&templates.StaticCtx{Logger: p.Logger}).RandomString(12),
	)
}
