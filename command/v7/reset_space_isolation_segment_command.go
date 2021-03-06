package v7

import (
	"code.cloudfoundry.org/cli/actor/sharedaction"
	"code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/actor/v7action"
	"code.cloudfoundry.org/cli/command"
	"code.cloudfoundry.org/cli/command/flag"
	sharedV2 "code.cloudfoundry.org/cli/command/v6/shared"
	"code.cloudfoundry.org/cli/command/v7/shared"
	"code.cloudfoundry.org/clock"
)

//go:generate counterfeiter . ResetSpaceIsolationSegmentActor

type ResetSpaceIsolationSegmentActor interface {
	ResetSpaceIsolationSegment(orgGUID string, spaceGUID string) (string, v7action.Warnings, error)
}

//go:generate counterfeiter . ResetSpaceIsolationSegmentActorV2

type ResetSpaceIsolationSegmentActorV2 interface {
	GetSpaceByOrganizationAndName(orgGUID string, spaceName string) (v2action.Space, v2action.Warnings, error)
}

type ResetSpaceIsolationSegmentCommand struct {
	RequiredArgs    flag.ResetSpaceIsolationArgs `positional-args:"yes"`
	usage           interface{}                  `usage:"CF_NAME reset-space-isolation-segment SPACE_NAME"`
	relatedCommands interface{}                  `related_commands:"org, restart, space"`

	UI          command.UI
	Config      command.Config
	SharedActor command.SharedActor
	Actor       ResetSpaceIsolationSegmentActor
	ActorV2     ResetSpaceIsolationSegmentActorV2
}

func (cmd *ResetSpaceIsolationSegmentCommand) Setup(config command.Config, ui command.UI) error {
	cmd.UI = ui
	cmd.Config = config
	cmd.SharedActor = sharedaction.NewActor(config)

	ccClient, _, err := shared.GetNewClientsAndConnectToCF(config, ui, "")
	if err != nil {
		return err
	}
	cmd.Actor = v7action.NewActor(ccClient, config, nil, nil, clock.NewClock())

	ccClientV2, uaaClientV2, err := sharedV2.GetNewClientsAndConnectToCF(config, ui)
	if err != nil {
		return err
	}
	cmd.ActorV2 = v2action.NewActor(ccClientV2, uaaClientV2, config)

	return nil
}

func (cmd ResetSpaceIsolationSegmentCommand) Execute(args []string) error {
	err := cmd.SharedActor.CheckTarget(true, false)
	if err != nil {
		return err
	}

	user, err := cmd.Config.CurrentUser()
	if err != nil {
		return err
	}

	cmd.UI.DisplayTextWithFlavor("Resetting isolation segment assignment of space {{.SpaceName}} in org {{.OrgName}} as {{.CurrentUser}}...", map[string]interface{}{
		"SpaceName":   cmd.RequiredArgs.SpaceName,
		"OrgName":     cmd.Config.TargetedOrganization().Name,
		"CurrentUser": user.Name,
	})

	space, v2Warnings, err := cmd.ActorV2.GetSpaceByOrganizationAndName(cmd.Config.TargetedOrganization().GUID, cmd.RequiredArgs.SpaceName)
	cmd.UI.DisplayWarningsV7(v2Warnings)
	if err != nil {
		return err
	}

	newIsolationSegmentName, warnings, err := cmd.Actor.ResetSpaceIsolationSegment(cmd.Config.TargetedOrganization().GUID, space.GUID)
	cmd.UI.DisplayWarningsV7(warnings)
	if err != nil {
		return err
	}

	cmd.UI.DisplayOK()

	if newIsolationSegmentName == "" {
		cmd.UI.DisplayText("Applications in this space will be placed in the platform default isolation segment.")
	} else {
		cmd.UI.DisplayText("Applications in this space will be placed in isolation segment {{.orgIsolationSegment}}.", map[string]interface{}{
			"orgIsolationSegment": newIsolationSegmentName,
		})
	}
	cmd.UI.DisplayText("Running applications need a restart to be moved there.")

	return nil
}
