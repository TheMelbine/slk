package demo

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

const msec = time.Millisecond

// scenarios maps each --demo argument to the rules its tape relies on.
// Every scenario also gets baseRules.
var scenarios = map[string]func() []Rule{
	"compose":    func() []Rule { return []Rule{answerTheUser()} },
	"hero":       heroRules,
	"reactions":  reactionRules,
	"themes":     func() []Rule { return nil },
	"threads":    threadRules,
	"workspaces": workspaceRules,
}

// ScenarioNames lists the valid scenarios, sorted.
func ScenarioNames() []string {
	names := slices.Collect(maps.Keys(scenarios))
	slices.Sort(names)
	return names
}

func rulesFor(name string) ([]Rule, error) {
	build, ok := scenarios[name]
	if !ok {
		return nil, fmt.Errorf("unknown demo scenario %q (choose from: %s)", name, strings.Join(ScenarioNames(), ", "))
	}
	return append(baseRules(), build()...), nil
}

// baseRules keep the sidebar and rail in step with the World.
func baseRules() []Rule {
	return []Rule{{
		When:  func(ev Event) bool { return ev.Kind == EventChannelOpened || ev.Kind == EventMarkedRead },
		Steps: []Step{{Act: readStateChanged}},
	}}
}

var replyTexts = []string{
	"Sounds good :thumbsup:",
	"Nice, thanks!",
	"On it :rocket:",
	"Love it :tada:",
}

// answerTheUser: whenever the user sends something, someone starts typing
// and then answers, in the thread if it was a thread reply.
func answerTheUser() Rule {
	return Rule{When: on(EventMessageSent, anyChannel), Steps: []Step{
		{After: time.Second, Act: responderTyping},
		{After: 2 * time.Second, Act: responderReply(replyTexts)},
	}}
}

// heroRules: Priya types and posts the first time #engineering opens and
// Sam reacts; the user gets answered; Driftwood OSS lights up in the rail.
// TestHeroEngineeringRule depends on the #engineering rule being first.
func heroRules() []Rule {
	return []Rule{
		{When: on(EventChannelOpened, chEngineering), Once: true, Steps: []Step{
			{After: 1500 * msec, Act: typing(uPriya, chEngineering)},
			{After: 2500 * msec, Act: post(uPriya, chEngineering, "Merged :tada: The flaky `TestSessionRefresh` is gone from CI.")},
			{After: 1200 * msec, Act: react(uSam, "rocket")},
		}},
		answerTheUser(),
		{When: on(EventStart, anyChannel), Once: true, Steps: []Step{
			{After: 4 * time.Second, Act: post(uOmar, chContributors, "CI on the plugin branch is green now :white_check_mark:")},
		}},
	}
}

// threadRules: Jonas replies live in the deploy thread once it is open.
func threadRules() []Rule {
	return []Rule{
		{When: on(EventThreadOpened, chDeploys), Once: true, Steps: []Step{
			{After: 1500 * msec, Act: typing(uJonas, chDeploys)},
			{After: 2 * time.Second, Act: reply(uJonas, "Dashboards look great. Nice rollout :clap:")},
		}},
		answerTheUser(),
	}
}

// reactionRules: three people react to the lunch message in #general.
func reactionRules() []Rule {
	return []Rule{
		{When: on(EventChannelOpened, chGeneral), Once: true, Steps: []Step{
			{After: 3 * time.Second, Act: reactLatest(uPriya, chGeneral, "taco")},
			{After: 800 * msec, Act: reactLatest(uSam, chGeneral, "taco")},
			{After: 800 * msec, Act: reactLatest(uTom, chGeneral, "heart_eyes")},
		}},
	}
}

// workspaceRules: Driftwood OSS lights up, and Ruth greets the user when
// they switch over.
func workspaceRules() []Rule {
	return []Rule{
		{When: on(EventStart, anyChannel), Once: true, Steps: []Step{
			{After: 3 * time.Second, Act: post(uOmar, chContributors, "CI on the plugin branch is green now :white_check_mark:")},
		}},
		{When: func(ev Event) bool { return ev.Kind == EventWorkspaceSwitched && ev.TeamID == teamDriftwood }, Once: true, Steps: []Step{
			{After: 1500 * msec, Act: typing(uRuth, chContributors)},
			{After: 2 * time.Second, Act: post(uRuth, chContributors, "Welcome back! The 0.9.1 checklist is pinned in <#C1ANNOUNCE> :pushpin:")},
		}},
	}
}
