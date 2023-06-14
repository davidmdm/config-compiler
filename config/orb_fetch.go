package config

import (
	"net/http"

	"github.com/CircleCI-Public/circleci-cli/api"
	"github.com/CircleCI-Public/circleci-cli/api/graphql"
)

var gql = graphql.NewClient(http.DefaultClient, "https://circleci.com", "graphql-unstable", "", false)

func GetOrbSource(ref string) (string, error) {
	return api.OrbSource(gql, ref)
}
