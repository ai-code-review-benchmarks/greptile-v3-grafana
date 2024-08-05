package zanzana

import (
	"fmt"
	"strconv"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
)

const (
	TypeUser      string = "user"
	TypeTeam      string = "team"
	TypeFolder    string = "folder"
	TypeDashboard string = "dashboard"
)

const (
	RelationTeamMember string = "member"
	RelationTeamAdmin  string = "admin"
	RelationParent     string = "parent"
)

func NewObject(typ, id, relation string) string {
	obj := fmt.Sprintf("%s:%s", typ, id)
	if relation != "" {
		obj = fmt.Sprintf("%s#%s", obj, relation)
	}
	return obj
}

func NewScopedObject(typ, id, relation, scope string) string {
	return NewObject(typ, fmt.Sprintf("%s-%s", scope, id), "")
}

func TranslateToTuple(user string, action, kind, identifier string, orgID int64) (*openfgav1.TupleKey, bool) {
	relation, ok := actionTranslations[action]
	if !ok {
		return nil, false
	}

	t, ok := kindTranslations[kind]
	if !ok {
		return nil, false
	}

	tuple := &openfgav1.TupleKey{
		Relation: relation,
	}

	tuple.User = user
	tuple.Relation = relation

	// Some uid:s in grafana are not guarantee to be unique across orgs so we need to scope them.
	if t.orgScoped {
		tuple.Object = NewScopedObject(t.typ, identifier, "", strconv.FormatInt(orgID, 10))
	} else {
		tuple.Object = NewObject(t.typ, identifier, "")
	}

	return tuple, true
}
