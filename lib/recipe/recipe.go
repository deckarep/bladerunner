package recipe

func NewRecipe() *BladeRecipe {
	return &BladeRecipe{
		Required:    &RequiredRecipe{},
		Overrides:   &OverridesRecipe{},
		Help:        &HelpRecipe{},
		Interaction: &InteractionRecipe{},
		Resilience:  &ResilienceRecipe{},
		Meta:        &MetaRecipe{},
	}
}

// // StepRecipe is an ordered series of recipes that will be attempted in the specified order.
// // The parameters specified in this recipe supercede the parameters in the individual recipe.
// type StepRecipe struct {
// 	// Hmmm...does a step recipe inherit the properties above? Or does it have it's own similar specialized properties.
// 	Recipe
// 	Steps             []*Recipe
// 	StepPauseDuration string
// }

// type AggregateRecipe struct {
// 	Recipe
// }

// BladeRecipe is the root recipe type.
type BladeRecipe struct {
	Required    *RequiredRecipe
	Overrides   *OverridesRecipe
	Help        *HelpRecipe
	Interaction *InteractionRecipe
	Resilience  *ResilienceRecipe
	Meta        *MetaRecipe
}

// This block is for prototyping a good design.
type RequiredRecipe struct {
	// Allows a single command to be executed for a given server session.
	Command string
	// Allows for a series of commands to be executed within a single server session.
	Commands          []string
	Hosts             []string `toml:"Hosts,omitempty"`             // Must specify one or the other.
	HostLookupCommand string   `toml:"HostLookupCommand,omitempty"` // What happens if both are?
}

type OverridesRecipe struct {
	Concurrency int
	User        string
	// HostLookupCacheDisabled indicates that you want HostLookupCommand's to never be cached based on global settings.
	HostLookupCacheDisabled bool
	// HostLookupCacheDuration specifies the amount of time to utilize cache before refreshing the host list.
	HostLookupCacheDuration string
}

type HelpRecipe struct {
	Short string
	Long  string
	Usage string
}

type InteractionRecipe struct {
	Banner       string
	PromptBanner bool
	PromptColor  string
}

type ResilienceRecipe struct {
	WaitDuration           string
	Retries                int
	RetryBackoffStrategy   string
	RetryBackoffMultiplier string // <-- this is a duration like 5s
	FailBatch              bool
}

type MetaRecipe struct {
	Name     string
	Filename string
}
