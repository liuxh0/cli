package v7

import (
	"sort"

	"code.cloudfoundry.org/cli/actor/sharedaction"
	"code.cloudfoundry.org/cli/actor/v7action"
	"code.cloudfoundry.org/cli/command"
	"code.cloudfoundry.org/cli/command/v7/shared"
	"code.cloudfoundry.org/cli/util/sorting"
	"code.cloudfoundry.org/cli/util/ui"
	"code.cloudfoundry.org/clock"
)

//go:generate counterfeiter . DomainsActor

type DomainsActor interface {
	GetOrganizationDomains(string, string) ([]v7action.Domain, v7action.Warnings, error)
}

type DomainsCommand struct {
	usage           interface{} `usage:"CF_NAME domains\n\nEXAMPLES:\n   CF_NAME domains\n   CF_NAME domains --labels 'environment in (production,staging),tier in (backend)'\n   CF_NAME domains --labels 'env=dev,!chargeback-code,tier in (backend,worker)'"`
	relatedCommands interface{} `related_commands:"create-private-domain, create-route, create-shared-domain, routes, set-label"`

	Labels      string `long:"labels" description:"Selector to filter domains by labels"`
	UI          command.UI
	Config      command.Config
	SharedActor command.SharedActor
	Actor       DomainsActor
}

func (cmd *DomainsCommand) Setup(config command.Config, ui command.UI) error {
	cmd.UI = ui
	cmd.Config = config
	cmd.SharedActor = sharedaction.NewActor(config)

	ccClient, _, err := shared.GetNewClientsAndConnectToCF(config, ui, "")
	if err != nil {
		return err
	}
	cmd.Actor = v7action.NewActor(ccClient, config, nil, nil, clock.NewClock())

	return nil
}

func (cmd DomainsCommand) Execute(args []string) error {
	err := cmd.SharedActor.CheckTarget(true, false)
	if err != nil {
		return err
	}

	currentUser, err := cmd.Config.CurrentUser()
	if err != nil {
		return err
	}

	targetedOrg := cmd.Config.TargetedOrganization()
	cmd.UI.DisplayTextWithFlavor("Getting domains in org {{.CurrentOrg}} as {{.CurrentUser}}...\n", map[string]interface{}{
		"CurrentOrg":  targetedOrg.Name,
		"CurrentUser": currentUser.Name,
	})

	domains, warnings, err := cmd.Actor.GetOrganizationDomains(targetedOrg.GUID, cmd.Labels)
	cmd.UI.DisplayWarningsV7(warnings)
	if err != nil {
		return err
	}

	sort.Slice(domains, func(i, j int) bool { return sorting.LessIgnoreCase(domains[i].Name, domains[j].Name) })

	if len(domains) > 0 {
		cmd.displayDomainsTable(domains)
	} else {
		cmd.UI.DisplayText("No domains found.")
	}
	return nil
}

func (cmd DomainsCommand) displayDomainsTable(domains []v7action.Domain) {
	var domainsTable = [][]string{
		{
			cmd.UI.TranslateText("name"),
			cmd.UI.TranslateText("availability"),
			cmd.UI.TranslateText("internal"),
		},
	}

	for _, domain := range domains {
		var availability string
		var internal string

		if domain.Shared() {
			availability = cmd.UI.TranslateText("shared")
		} else {
			availability = cmd.UI.TranslateText("private")
		}

		if domain.Internal.IsSet && domain.Internal.Value {
			internal = cmd.UI.TranslateText("true")
		}

		domainsTable = append(domainsTable, []string{
			domain.Name,
			availability,
			internal,
		})
	}

	cmd.UI.DisplayTableWithHeader("", domainsTable, ui.DefaultTableSpacePadding)

}
