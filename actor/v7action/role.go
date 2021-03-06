package v7action

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccerror"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3/constant"
)

type Role ccv3.Role

func (actor Actor) CreateOrgRole(roleType constant.RoleType, orgGUID string, userNameOrGUID string, userOrigin string, isClient bool) (Warnings, error) {
	roleToCreate := ccv3.Role{
		Type:    roleType,
		OrgGUID: orgGUID,
	}

	if isClient {
		roleToCreate.UserGUID = userNameOrGUID
	} else {
		roleToCreate.Username = userNameOrGUID
		roleToCreate.Origin = userOrigin
	}

	_, warnings, err := actor.CloudControllerClient.CreateRole(roleToCreate)

	return Warnings(warnings), err
}

func (actor Actor) CreateSpaceRole(roleType constant.RoleType, orgGUID string, spaceGUID string, userNameOrGUID string, userOrigin string, isClient bool) (Warnings, error) {
	roleToCreate := ccv3.Role{
		Type:      roleType,
		SpaceGUID: spaceGUID,
	}

	if isClient {
		roleToCreate.UserGUID = userNameOrGUID
	} else {
		roleToCreate.Username = userNameOrGUID
		roleToCreate.Origin = userOrigin
	}

	warnings, err := actor.CreateOrgRole(constant.OrgUserRole, orgGUID, userNameOrGUID, userOrigin, isClient)
	if err != nil {
		if _, isIdempotentError := err.(ccerror.RoleAlreadyExistsError); !isIdempotentError {
			return warnings, err
		}
	}

	_, ccv3Warnings, err := actor.CloudControllerClient.CreateRole(roleToCreate)
	warnings = append(warnings, ccv3Warnings...)

	return warnings, err
}

func (actor Actor) GetOrgUsersByRoleType(orgGuid string) (map[constant.RoleType][]User, Warnings, error) {
	return actor.getUsersByRoleType(orgGuid, ccv3.OrganizationGUIDFilter)
}

func (actor Actor) GetSpaceUsersByRoleType(spaceGuid string) (map[constant.RoleType][]User, Warnings, error) {
	return actor.getUsersByRoleType(spaceGuid, ccv3.SpaceGUIDFilter)
}

func (actor Actor) getUsersByRoleType(guid string, filterKey ccv3.QueryKey) (map[constant.RoleType][]User, Warnings, error) {
	ccv3Roles, includes, ccWarnings, err := actor.CloudControllerClient.GetRoles(
		ccv3.Query{
			Key:    filterKey,
			Values: []string{guid},
		},
		ccv3.Query{
			Key:    ccv3.Include,
			Values: []string{"user"},
		},
	)
	if err != nil {
		return nil, Warnings(ccWarnings), err
	}
	usersByGuids := make(map[string]ccv3.User)
	for _, user := range includes.Users {
		usersByGuids[user.GUID] = user
	}
	usersByRoleType := make(map[constant.RoleType][]User)
	for _, role := range ccv3Roles {
		user := User(usersByGuids[role.UserGUID])
		usersByRoleType[role.Type] = append(usersByRoleType[role.Type], user)
	}
	return usersByRoleType, Warnings(ccWarnings), nil
}
