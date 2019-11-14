package v7action

import (
	"sort"
	"strings"

	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccerror"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3/constant"
	"code.cloudfoundry.org/cli/util/sorting"
)

type RouteDestination struct {
	GUID string
	App  RouteDestinationApp
}

type RouteDestinationApp ccv3.RouteDestinationApp

type Route struct {
	GUID       string
	SpaceGUID  string
	DomainGUID string
	Host       string
	Path       string
	DomainName string
	SpaceName  string
	URL        string
}

type RouteSummary struct {
	Route
	AppNames []string
}

func (actor Actor) CreateRoute(spaceGUID, domainName, hostname, path string) (Route, Warnings, error) {
	allWarnings := Warnings{}
	domain, warnings, err := actor.GetDomainByName(domainName)
	allWarnings = append(allWarnings, warnings...)

	if err != nil {
		return Route{}, allWarnings, err
	}

	route, apiWarnings, err := actor.CloudControllerClient.CreateRoute(ccv3.Route{
		SpaceGUID:  spaceGUID,
		DomainGUID: domain.GUID,
		Host:       hostname,
		Path:       path,
	})

	actorWarnings := Warnings(apiWarnings)
	allWarnings = append(allWarnings, actorWarnings...)

	if _, ok := err.(ccerror.RouteNotUniqueError); ok {
		return Route{}, allWarnings, actionerror.RouteAlreadyExistsError{Err: err}
	}

	return Route{
		GUID:       route.GUID,
		Host:       route.Host,
		Path:       route.Path,
		SpaceGUID:  route.SpaceGUID,
		DomainGUID: route.DomainGUID,
		SpaceName:  spaceGUID,
		DomainName: domainName,
	}, allWarnings, err
}

func (actor Actor) GetRouteDestinations(routeGUID string) ([]RouteDestination, Warnings, error) {
	destinations, warnings, err := actor.CloudControllerClient.GetRouteDestinations(routeGUID)

	actorDestinations := []RouteDestination{}
	for _, dst := range destinations {
		actorDestinations = append(actorDestinations, RouteDestination{
			GUID: dst.GUID,
			App:  RouteDestinationApp(dst.App),
		})
	}

	return actorDestinations, Warnings(warnings), err
}

func (actor Actor) GetRouteDestinationByAppGUID(routeGUID string, appGUID string) (RouteDestination, Warnings, error) {
	allDestinations, warnings, err := actor.GetRouteDestinations(routeGUID)
	if err != nil {
		return RouteDestination{}, warnings, err
	}

	for _, destination := range allDestinations {
		if destination.App.GUID == appGUID && destination.App.Process.Type == constant.ProcessTypeWeb {
			return destination, warnings, nil
		}
	}

	return RouteDestination{}, warnings, actionerror.RouteDestinationNotFoundError{
		AppGUID:     appGUID,
		ProcessType: constant.ProcessTypeWeb,
		RouteGUID:   routeGUID,
	}
}

func (actor Actor) GetRoutesBySpace(spaceGUID string) ([]Route, Warnings, error) {
	allWarnings := Warnings{}

	routes, warnings, err := actor.CloudControllerClient.GetRoutes(
		ccv3.Query{
			Key:    ccv3.SpaceGUIDFilter,
			Values: []string{spaceGUID},
		},
	)
	allWarnings = append(allWarnings, warnings...)
	if err != nil {
		return nil, allWarnings, err
	}

	ret, actor_warnings, err := actor.createActionRoutes(routes, allWarnings)

	return ret, actor_warnings, err
}

func (actor Actor) parseRoutePath(routePath string) (string, string, string, string, Warnings, error) {
	var warnings Warnings
	var hostPart = ""
	var pathPart = ""
	routeParts := strings.SplitN(routePath, "/", 2)
	domainName := routeParts[0]
	if len(routeParts) > 1 {
		pathPart = "/" + routeParts[1]
	}
	domainParts := strings.SplitN(domainName, ".", 2)
	domainHasMultipleParts := len(domainParts) > 1

	domain, allWarnings, err := actor.GetDomainByName(domainName)

	_, domainNotFound := err.(actionerror.DomainNotFoundError)

	needToCheckForHost := domainNotFound && domainHasMultipleParts
	if err != nil && !needToCheckForHost {
		return "", "", "", "", allWarnings, err
	}

	if needToCheckForHost {
		domainName = domainParts[1]
		hostPart = domainParts[0]
		domain, warnings, err = actor.GetDomainByName(domainName)
		allWarnings = append(allWarnings, warnings...)
		if err != nil {
			return "", "", "", "", allWarnings, err
		}
	}

	return hostPart, pathPart, domainName, domain.GUID, allWarnings, nil
}

func (actor Actor) GetRouteByNameAndSpace(routePath string, spaceGUID string) (Route, Warnings, error) {
	filters := []ccv3.Query{
		ccv3.Query{
			Key:    ccv3.SpaceGUIDFilter,
			Values: []string{spaceGUID},
		},
	}
	hostPart, pathPart, domainName, domainGUID, allWarnings, err := actor.parseRoutePath(routePath)
	if err != nil {
		return Route{}, allWarnings, err
	}
	filters = append(filters, ccv3.Query{
		Key:    ccv3.DomainGUIDFilter,
		Values: []string{domainGUID},
	})
	filters = append(filters, ccv3.Query{Key: ccv3.HostsFilter,
		Values: []string{hostPart},
	})
	filters = append(filters, ccv3.Query{Key: ccv3.PathsFilter,
		Values: []string{pathPart},
	})
	routes, warnings, err := actor.CloudControllerClient.GetRoutes(filters...)
	allWarnings = append(allWarnings, warnings...)
	if err != nil {
		return Route{}, allWarnings, err
	}
	if len(routes) == 0 {
		return Route{}, allWarnings, actionerror.RouteNotFoundError{
			Host:       hostPart,
			DomainName: domainName,
			Path:       pathPart,
		}
	}
	actionRoutes, allWarnings, err := actor.createActionRoutes(routes, allWarnings)
	if err != nil {
		return Route{}, allWarnings, err
	}
	return actionRoutes[0], allWarnings, nil
}

func (actor Actor) GetRoutesByOrg(orgGUID string) ([]Route, Warnings, error) {
	allWarnings := Warnings{}

	routes, warnings, err := actor.CloudControllerClient.GetRoutes(ccv3.Query{
		Key:    ccv3.OrganizationGUIDFilter,
		Values: []string{orgGUID},
	})
	allWarnings = append(allWarnings, warnings...)
	if err != nil {
		return nil, allWarnings, err
	}

	return actor.createActionRoutes(routes, allWarnings)
}

func (actor Actor) GetRouteSummaries(routes []Route) ([]RouteSummary, Warnings, error) {
	var allWarnings Warnings
	var routeSummaries []RouteSummary

	destinationAppGUIDsByRouteGUID := make(map[string][]string)
	destinationAppGUIDs := make(map[string]bool)
	var uniqueAppGUIDs []string

	for _, route := range routes {
		destinations, warnings, err := actor.GetRouteDestinations(route.GUID)
		allWarnings = append(allWarnings, warnings...)
		if err != nil {
			return nil, allWarnings, err
		}

		for _, destination := range destinations {
			appGUID := destination.App.GUID

			if _, ok := destinationAppGUIDs[appGUID]; !ok {
				destinationAppGUIDs[appGUID] = true
				uniqueAppGUIDs = append(uniqueAppGUIDs, appGUID)
			}

			destinationAppGUIDsByRouteGUID[route.GUID] = append(destinationAppGUIDsByRouteGUID[route.GUID], appGUID)
		}
	}

	apps, warnings, err := actor.GetApplicationsByGUIDs(uniqueAppGUIDs)
	allWarnings = append(allWarnings, warnings...)
	if err != nil {
		return nil, allWarnings, err
	}

	appNamesByGUID := make(map[string]string)
	for _, app := range apps {
		appNamesByGUID[app.GUID] = app.Name
	}

	for _, route := range routes {
		var appNames []string

		appGUIDs := destinationAppGUIDsByRouteGUID[route.GUID]
		for _, appGUID := range appGUIDs {
			appNames = append(appNames, appNamesByGUID[appGUID])
		}

		routeSummaries = append(routeSummaries, RouteSummary{
			Route:    route,
			AppNames: appNames,
		})
	}

	sort.Slice(routeSummaries, func(i, j int) bool {
		return sorting.LessIgnoreCase(routeSummaries[i].SpaceName, routeSummaries[j].SpaceName)
	})

	return routeSummaries, allWarnings, nil
}

func (actor Actor) DeleteOrphanedRoutes(spaceGUID string) (Warnings, error) {
	var allWarnings Warnings

	jobURL, warnings, err := actor.CloudControllerClient.DeleteOrphanedRoutes(spaceGUID)
	allWarnings = append(allWarnings, warnings...)
	if err != nil {
		return allWarnings, err
	}

	warnings, err = actor.CloudControllerClient.PollJob(jobURL)
	allWarnings = append(allWarnings, warnings...)

	return allWarnings, err
}

func (actor Actor) DeleteRoute(domainName, hostname, path string) (Warnings, error) {
	allWarnings := Warnings{}
	domain, warnings, err := actor.GetDomainByName(domainName)
	allWarnings = append(allWarnings, warnings...)

	if _, ok := err.(actionerror.DomainNotFoundError); ok {
		allWarnings = append(allWarnings, err.Error())
		return allWarnings, nil
	}

	if err != nil {
		return allWarnings, err
	}

	queryArray := []ccv3.Query{
		{Key: ccv3.DomainGUIDFilter, Values: []string{domain.GUID}},
		{Key: ccv3.HostsFilter, Values: []string{hostname}},
		{Key: ccv3.PathsFilter, Values: []string{path}},
	}

	routes, apiWarnings, err := actor.CloudControllerClient.GetRoutes(queryArray...)

	actorWarnings := Warnings(apiWarnings)
	allWarnings = append(allWarnings, actorWarnings...)

	if err != nil {
		return allWarnings, err
	}

	if len(routes) == 0 {
		return allWarnings, actionerror.RouteNotFoundError{
			DomainName: domainName,
			Host:       hostname,
			Path:       path,
		}
	}

	jobURL, apiWarnings, err := actor.CloudControllerClient.DeleteRoute(routes[0].GUID)
	actorWarnings = Warnings(apiWarnings)
	allWarnings = append(allWarnings, actorWarnings...)

	if err != nil {
		return allWarnings, err
	}

	pollJobWarnings, err := actor.CloudControllerClient.PollJob(jobURL)
	allWarnings = append(allWarnings, Warnings(pollJobWarnings)...)

	return allWarnings, err
}

func (actor Actor) GetRouteByAttributes(domainName string, domainGUID string, hostname string, path string) (Route, Warnings, error) {
	ccRoutes, ccWarnings, err := actor.CloudControllerClient.GetRoutes(
		ccv3.Query{Key: ccv3.DomainGUIDFilter, Values: []string{domainGUID}},
		ccv3.Query{Key: ccv3.HostsFilter, Values: []string{hostname}},
		ccv3.Query{Key: ccv3.PathsFilter, Values: []string{path}},
	)

	if err != nil {
		return Route{}, Warnings(ccWarnings), err
	}

	if len(ccRoutes) < 1 {
		return Route{}, Warnings(ccWarnings), actionerror.RouteNotFoundError{
			DomainName: domainName,
			DomainGUID: domainGUID,
			Host:       hostname,
			Path:       path,
		}
	}

	return Route{
		GUID:       ccRoutes[0].GUID,
		Host:       ccRoutes[0].Host,
		Path:       ccRoutes[0].Path,
		SpaceGUID:  ccRoutes[0].SpaceGUID,
		DomainGUID: ccRoutes[0].DomainGUID,
	}, Warnings(ccWarnings), nil
}

func (actor Actor) MapRoute(routeGUID string, appGUID string) (Warnings, error) {
	warnings, err := actor.CloudControllerClient.MapRoute(routeGUID, appGUID)
	return Warnings(warnings), err
}

func (actor Actor) UnmapRoute(routeGUID string, destinationGUID string) (Warnings, error) {
	warnings, err := actor.CloudControllerClient.UnmapRoute(routeGUID, destinationGUID)
	return Warnings(warnings), err
}

func (actor Actor) GetApplicationRoutes(appGUID string) ([]Route, Warnings, error) {
	allWarnings := Warnings{}

	routes, warnings, err := actor.CloudControllerClient.GetApplicationRoutes(appGUID)
	allWarnings = append(allWarnings, warnings...)
	if err != nil {
		return nil, allWarnings, err
	}

	if len(routes) == 0 {
		return nil, allWarnings, err
	}

	return actor.createActionRoutes(routes, allWarnings)
}

func (actor Actor) createActionRoutes(routes []ccv3.Route, allWarnings Warnings) ([]Route, Warnings, error) {
	spaceGUIDsSet := map[string]struct{}{}
	spacesQuery := ccv3.Query{Key: ccv3.GUIDFilter, Values: []string{}}

	for _, route := range routes {
		if _, ok := spaceGUIDsSet[route.SpaceGUID]; !ok {
			spacesQuery.Values = append(spacesQuery.Values, route.SpaceGUID)
			spaceGUIDsSet[route.SpaceGUID] = struct{}{}
		}
	}

	spaces, warnings, err := actor.CloudControllerClient.GetSpaces(spacesQuery)
	allWarnings = append(allWarnings, warnings...)
	if err != nil {
		return nil, allWarnings, err
	}

	spacesByGUID := map[string]ccv3.Space{}
	for _, space := range spaces {
		spacesByGUID[space.GUID] = space
	}

	actorRoutes := []Route{}
	for _, route := range routes {
		actorRoutes = append(actorRoutes, Route{
			GUID:       route.GUID,
			Host:       route.Host,
			Path:       route.Path,
			SpaceGUID:  route.SpaceGUID,
			DomainGUID: route.DomainGUID,
			URL:        route.URL,
			SpaceName:  spacesByGUID[route.SpaceGUID].Name,
			DomainName: getDomainName(route.URL, route.Host, route.Path),
		})
	}

	return actorRoutes, allWarnings, nil
}

func getDomainName(fullURL, host, path string) string {
	domainWithoutHost := strings.TrimPrefix(fullURL, host+".")
	return strings.TrimSuffix(domainWithoutHost, path)
}
